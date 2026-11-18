//go:build integration

package app

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/database"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func integrationConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load("../../configs")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Env != "test" || cfg.Database.DBName != "stars_admin_test" {
		t.Fatal("必须使用隔离测试环境")
	}
	cfg.Redis.KeyPrefix = "stars:test:" + uuid.New().String() + ":"
	return cfg
}

func TestRealRedisConcurrencyAndDependencyFailure(t *testing.T) {
	cfg := integrationConfig(t)
	if err := database.Migrate(t.Context(), cfg, "up", firstPassword); err != nil {
		t.Fatal(err)
	}
	cfg.RateLimit.Requests = 20
	a, err := New(t.Context(), cfg, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Close(); err != nil && !errors.Is(err, redis.ErrClosed) {
			t.Error(err)
		}
	})
	if err := a.Ready(t.Context()); err != nil {
		t.Fatal(err)
	}
	var allowed, rejected atomic.Int64
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			switch request(a.router, "GET", "/api/ping").Code {
			case 200:
				allowed.Add(1)
			case 429:
				rejected.Add(1)
			default:
				t.Error("真实 Redis 并发限流返回异常")
			}
		})
	}
	wg.Wait()
	if allowed.Load() != 20 || rejected.Load() != 80 {
		t.Fatalf("并发窗口未原子执行：成功 %d，拒绝 %d", allowed.Load(), rejected.Load())
	}
	keys, _, err := a.cache.Scan(t.Context(), 0, cfg.Redis.KeyPrefix+"limit:*", 100).Result()
	if err != nil || len(keys) != 1 {
		t.Fatal("限流键数量异常")
	}
	if ttl := a.cache.PTTL(t.Context(), keys[0]).Val(); ttl <= 0 || ttl > cfg.RateLimit.Window {
		t.Fatal("限流键缺少有效 TTL")
	}
	if err := a.cache.Close(); err != nil {
		t.Fatal(err)
	}
	if request(a.router, "GET", "/api/health/ready").Code != 503 || request(a.router, "GET", "/api/health/live").Code != 200 || request(a.router, "GET", "/api/ping").Code != 503 {
		t.Fatal("依赖故障时的存活、就绪或限流行为错误")
	}
}

func TestInitializationFailureClosesDatabase(t *testing.T) {
	cfg := integrationConfig(t)
	probe, err := database.Open(t.Context(), cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = probe.Close() }()
	count := func() int {
		var n int
		if err := probe.SQL.QueryRowContext(t.Context(), "SELECT count(*) FROM pg_stat_activity WHERE application_name='stars-admin'").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := count()
	cfg.Redis.Addr = "127.0.0.1:1"
	cfg.Redis.Timeout = 20 * time.Millisecond
	if _, err := New(t.Context(), cfg, zap.NewNop()); err == nil {
		t.Fatal("Redis 初始化失败不能启动")
	}
	deadline := time.Now().Add(time.Second)
	for count() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if count() > before {
		t.Fatal("初始化失败遗留 PostgreSQL 连接")
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := database.CheckMigrations(ctx, probe.SQL); err != nil {
		t.Fatal("Redis 不可用不应影响数据库版本检查")
	}
}

func TestDatabaseFailureKeepsLiveness(t *testing.T) {
	cfg := integrationConfig(t)
	if err := database.Migrate(t.Context(), cfg, "up", firstPassword); err != nil {
		t.Fatal(err)
	}
	a, err := New(t.Context(), cfg, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := a.Ready(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := a.db.Close(); err != nil {
		t.Fatal(err)
	}
	if request(a.router, "GET", "/api/health/ready").Code != 503 || request(a.router, "GET", "/api/health/live").Code != 200 {
		t.Fatal("数据库故障不得使存活与就绪检查混淆")
	}
}
