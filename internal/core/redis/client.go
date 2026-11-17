// Package redis 管理 Redis 连接和连通检查。
package redis

import (
	"context"
	"fmt"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	goredis "github.com/redis/go-redis/v9"
)

// Open 建立有明确超时的 Redis 客户端，失败时关闭连接池。
func Open(ctx context.Context, c config.Redis) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr: c.Addr, Password: c.Password, DB: c.DB, PoolSize: c.PoolSize,
		DialTimeout: c.Timeout, ReadTimeout: c.Timeout, WriteTimeout: c.Timeout,
		ContextTimeoutEnabled: true, MaxRetries: -1,
	})
	probe, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	if err := client.Ping(probe).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("检查 Redis 连通性失败: %w", err)
	}
	return client, nil
}
