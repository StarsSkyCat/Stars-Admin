// Package middleware 提供 HTTP 横切能力，不依赖业务模块。
package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const requestIDKey = "requestID"

// RequestID 为每个请求生成独立标识，不信任客户端注入值。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.New().String()
		c.Set(requestIDKey, id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

// AccessLog 只记录固定白名单元信息，不采集请求头、查询参数和请求体。
func AccessLog(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = "<未匹配路由>"
		}
		log.Info("HTTP 请求", zap.String("traceId", c.GetString(requestIDKey)),
			zap.String("method", c.Request.Method), zap.String("route", path),
			zap.Int("status", c.Writer.Status()), zap.Duration("duration", time.Since(started)))
	}
}

// Recovery 捕获 panic，仅在服务端记录堆栈，不记录可能包含敏感数据的 panic 值。
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recover() != nil {
				log.Error("请求发生 panic", zap.String("traceId", c.GetString(requestIDKey)), zap.ByteString("stack", debug.Stack()))
				if !c.Writer.Written() {
					response.Fail(c, response.ErrInternal)
				} else {
					c.Abort()
				}
			}
		}()
		c.Next()
	}
}

// CORS 使用来源白名单，不允许跨域凭据。
func CORS(origins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		c.Writer.Header().Add("Vary", "Origin")
		if slices.Contains(origins, "*") {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if slices.Contains(origins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			response.Fail(c, response.ErrForbidden)
			return
		}
		c.Header("Access-Control-Expose-Headers", "X-Request-Id")
		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Evaler 是限流所需的最小 Redis 能力。
type Evaler interface {
	Eval(context.Context, string, []string, ...any) *redis.Cmd
}

const increment = `local n = redis.call('INCR', KEYS[1]); if n == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]); end; return n`

// RateLimit 使用 Lua 原子设置固定窗口，依赖故障时拒绝放行。
func RateLimit(client Evaler, cfg config.RateLimit, cache config.Redis, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if !cfg.Enabled || path == "" || path == "/api/health" || strings.HasPrefix(path, "/api/health/") || strings.HasPrefix(path, "/swagger/") {
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), cache.Timeout)
		defer cancel()
		key := cache.KeyPrefix + "limit:ip:" + c.ClientIP() + ":" + c.Request.Method + ":" + path
		count, err := client.Eval(ctx, increment, []string{key}, cfg.Window.Milliseconds()).Int64()
		if err != nil {
			log.Warn("限流依赖暂不可用", zap.String("traceId", c.GetString(requestIDKey)))
			response.Fail(c, response.ErrUnavailable)
			return
		}
		if count > int64(cfg.Requests) {
			c.Header("Retry-After", strconv.FormatInt(max(1, int64(cfg.Window.Seconds()+0.999)), 10))
			response.Fail(c, response.ErrRateLimited)
			return
		}
		c.Next()
	}
}
