package app

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type fakeRedis struct {
	count  atomic.Int64
	failed bool
}

func (f *fakeRedis) Eval(context.Context, string, []string, ...any) *redis.Cmd {
	if f.failed {
		return redis.NewCmdResult(nil, errors.New("secret redis error"))
	}
	return redis.NewCmdResult(f.count.Add(1), nil)
}

func routerConfig(t *testing.T) *config.Config {
	t.Helper()
	c, err := config.Load("../../configs")
	if err != nil {
		t.Fatal(err)
	}
	c.RateLimit = config.RateLimit{Enabled: true, Requests: 2, Window: time.Minute}
	c.Redis.Timeout = time.Second
	return c
}

func request(router http.Handler, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestPingRoutesAndReadiness(t *testing.T) {
	cfg := routerConfig(t)
	router, err := NewRouter(cfg, zap.NewNop(), &fakeRedis{}, func(context.Context) error { return errors.New("database password=secret") })
	if err != nil {
		t.Fatal(err)
	}
	w := request(router, "GET", "/api/ping")
	if w.Code != 200 {
		t.Fatalf("ping 错误：%s", w.Body.String())
	}
	if _, err := uuid.Parse(w.Header().Get("X-Request-Id")); err != nil {
		t.Fatal("缺少有效请求标识")
	}
	var body response.Body[response.Status]
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != 0 || body.Data.Status != "ok" {
		t.Fatal("ping 响应不符合约定")
	}
	for _, test := range []struct {
		path, method string
		status       int
	}{
		{"/api/health/live", "GET", 200}, {"/api/health/ready", "GET", 503},
		{"/api/health", "GET", 503}, {"/missing", "GET", 404}, {"/api/ping", "POST", 405},
	} {
		w := request(router, test.method, test.path)
		if w.Code != test.status || strings.Contains(w.Body.String(), "secret") {
			t.Fatalf("路由行为错误：%s %d", test.path, w.Code)
		}
	}
}

func TestLimiterAndDependencyFailure(t *testing.T) {
	cfg := routerConfig(t)
	router, err := NewRouter(cfg, zap.NewNop(), &fakeRedis{}, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []int{200, 200, 429} {
		if got := request(router, "GET", "/api/ping").Code; got != expected {
			t.Fatalf("限流状态：%d，期望 %d", got, expected)
		}
	}
	if request(router, "GET", "/api/health/live").Code != 200 {
		t.Fatal("探针不应被限流")
	}
	router, err = NewRouter(cfg, zap.NewNop(), &fakeRedis{failed: true}, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if request(router, "GET", "/api/ping").Code != 503 {
		t.Fatal("Redis 故障不得绕过限流")
	}
}

func TestCORSRecoveryAndLogRedaction(t *testing.T) {
	cfg := routerConfig(t)
	cfg.CORS.Origins = []string{"https://allowed.example"}
	cfg.RateLimit.Enabled = false
	core, recorded := observer.New(zap.InfoLevel)
	router, err := NewRouter(cfg, zap.New(core), &fakeRedis{}, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/panic", func(*gin.Context) { panic("password=secret-token") })
	w := request(router, "GET", "/panic?token=secret-query")
	if w.Code != 500 || strings.Contains(w.Body.String(), "secret") {
		t.Fatal("panic 响应泄露内部信息")
	}
	for _, entry := range recorded.All() {
		for _, value := range entry.ContextMap() {
			text := fmt.Sprint(value)
			if bytes, ok := value.([]byte); ok {
				text = string(bytes)
			}
			if strings.Contains(text, "secret-token") || strings.Contains(text, "secret-query") {
				t.Fatal("日志采集敏感内容")
			}
		}
	}
	for _, test := range []struct {
		origin string
		status int
	}{{"https://allowed.example", 204}, {"https://denied.example", 403}} {
		r := httptest.NewRequest("OPTIONS", "/api/ping", nil)
		r.Header.Set("Origin", test.origin)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != test.status || w.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Fatal("CORS 白名单或凭据设置错误")
		}
	}
}

func TestSwaggerSwitch(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		cfg := routerConfig(t)
		cfg.Swagger.Enabled = enabled
		router, err := NewRouter(cfg, zap.NewNop(), &fakeRedis{}, func(context.Context) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		w := request(router, "GET", "/swagger/index.html")
		if enabled && w.Code != 200 || !enabled && w.Code != 404 {
			t.Fatal("Swagger 开关未生效")
		}
		if enabled {
			w = request(router, "GET", "/swagger/doc.json")
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"/ping"`) {
				t.Fatal("Swagger 缺少 ping 文档")
			}
		}
	}
}
