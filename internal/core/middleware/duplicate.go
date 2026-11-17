package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
	"uuid"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// DuplicateSubmit 以主体、路由和业务目标互斥关键写入，随机所有者避免误解锁。
func DuplicateSubmit(client *redis.Client, prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := httpx.Actor(c)
		sum := sha256.Sum256([]byte(strconv.FormatInt(int64(p.ID), 10) + ":" + c.Request.Method + ":" + c.FullPath() + ":" + c.Param("id")))
		key := prefix + "lock:submit:" + hex.EncodeToString(sum[:])
		owner := uuid.New().String()
		ok, err := client.SetNX(c.Request.Context(), key, owner, 30*time.Second).Result()
		if err != nil {
			response.Fail(c, response.ErrUnavailable)
			return
		}
		if !ok {
			response.Fail(c, response.ErrDuplicateSubmit)
			return
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = client.Eval(ctx, `if redis.call('GET',KEYS[1])==ARGV[1] then return redis.call('DEL',KEYS[1]) end;return 0`, []string{key}, owner).Err()
		}()
		c.Next()
	}
}
