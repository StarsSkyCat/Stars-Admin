// Package cache 提供有界的普通缓存操作，不用于认证或授权放行。
package cache

import (
	"context"
	"encoding/json/v2"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Store 只处理普通 TTL 缓存的序列化及失效重试。
type Store struct {
	client *redis.Client
	prefix string
	log    *zap.Logger
}

// New 创建由服务层调用的缓存工具。
func New(c *redis.Client, prefix string, log *zap.Logger) *Store { return &Store{c, prefix, log} }

// Read 缓存不可用时返回未命中，调用者仍须查询数据库。
func (s *Store) Read(ctx context.Context, key string, out any) bool {
	raw, err := s.client.Get(ctx, s.prefix+key).Bytes()
	return err == nil && json.Unmarshal(raw, out) == nil
}

// Put 尽力回填，普通缓存失败不改变数据库结果。
func (s *Store) Put(ctx context.Context, key string, v any) {
	raw, err := json.Marshal(v)
	if err == nil {
		if err = s.client.Set(ctx, s.prefix+key, raw, 30*time.Minute).Err(); err != nil {
			s.log.Warn("普通缓存回填失败")
		}
	}
}

// Invalidate 在提交后使用独立超时有界重试，耗尽后告警；TTL 限制陈旧窗口。
func (s *Store) Invalidate(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for range 3 {
		if s.client.Del(ctx, s.prefix+key).Err() == nil {
			return
		}
	}
	s.log.Error("数据库已提交，缓存失效重试耗尽", zap.String("cacheKey", key))
}

// Refresh 使用 SCAN 批量失效本功能命名空间，不使用 KEYS。
func (s *Store) Refresh(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, next, err := s.client.Scan(ctx, cursor, s.prefix+pattern, 100).Result()
		if err != nil {
			return err
		}
		for _, key := range keys {
			if err := s.client.Del(ctx, key).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}
