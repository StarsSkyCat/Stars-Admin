//go:build integration

// Package testutil 只用于隔离 PostgreSQL/Redis 集成测试。
package testutil

import (
	"testing"
	"uuid"

	"github.com/StarsSkyCat/Stars-Admin/internal/core"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/database"
	coreredis "github.com/StarsSkyCat/Stars-Admin/internal/core/redis"
	"github.com/StarsSkyCat/Stars-Admin/pkg/snowflake"
	"go.uber.org/zap"
)

// Open 允许不同目录的测试传入已加载配置，仍严格核对隔离库。
func Open(t *testing.T, cfg *config.Config) core.Deps {
	t.Helper()
	if cfg.App.Env != "test" || cfg.Database.DBName != "stars_admin_test" {
		t.Fatal("必须使用隔离测试库")
	}
	cfg.Redis.KeyPrefix = "stars:test:" + uuid.New().String() + ":"
	cfg.Auth.BcryptCost = 4
	if err := database.Migrate(t.Context(), cfg, "up", "Integration-Initial-Password"); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(t.Context(), cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	cache, err := coreredis.Open(t.Context(), cfg.Redis)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := cache.Close(); err != nil {
			t.Error(err)
		}
	})
	ids, err := snowflake.New(cfg.ID.Node)
	if err != nil {
		t.Fatal(err)
	}
	return core.Deps{Config: cfg, DB: db.ORM, Redis: cache, Logger: zap.NewNop(), IDs: ids}
}
