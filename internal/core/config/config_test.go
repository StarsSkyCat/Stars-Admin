package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, env string) string {
	t.Helper()
	// 隔离 Compose 为真实数据库设置的环境变量，测试配置优先级本身。
	for _, entry := range os.Environ() {
		key, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "STARS_") {
			if err := os.Unsetenv(key); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.Setenv(key, value); err != nil {
					t.Error(err)
				}
			})
		}
	}
	t.Setenv("APP_ENV", env)
	dir := t.TempDir()
	for _, name := range []string{"config.yaml", "config.prod.yaml", "config.test.yaml"} {
		data, err := os.ReadFile(filepath.Join("../../../configs", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestEnvironmentPriorityAndEmptyValues(t *testing.T) {
	dir := fixture(t, "prod")
	t.Setenv("STARS_DATABASE_HOST", "override.example")
	t.Setenv("STARS_DATABASE_PASSWORD", "")
	t.Setenv("STARS_LOG_FILE", "")
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.App.Env != "prod" || c.Database.Host != "override.example" || c.Database.Password != "" || c.Log.File != "" || c.Log.Format != "json" || c.Swagger.Enabled {
		t.Fatal("环境覆盖、空值覆盖或生产默认值不符合约定")
	}
	if err := c.ValidateServer(); err == nil {
		t.Fatal("生产空密码不应通过")
	}
}

func TestProductionValidationAndMigrationIsolation(t *testing.T) {
	dir := fixture(t, "prod")
	t.Setenv("STARS_DATABASE_PASSWORD", "production-test-password")
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateDatabase(); err != nil {
		t.Fatalf("迁移不应依赖 JWT: %v", err)
	}
	if err := c.ValidateServer(); err == nil {
		t.Fatal("HTTP 缺少 JWT 密钥应失败")
	}
	t.Setenv("STARS_JWT_SECRET", strings.Repeat("x", 32))
	c, err = Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateServer(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STARS_DATABASE_PASSWORD", "stars123")
	c, err = Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateDatabase(); err == nil {
		t.Fatal("生产开发默认密码不应通过")
	}
}

func TestRejectInvalidConfiguration(t *testing.T) {
	for _, test := range []struct{ name, key, value string }{
		{"非法环境", "APP_ENV", "../prod"},
		{"空主机", "STARS_DATABASE_HOST", ""},
		{"开启自动迁移", "STARS_DATABASE_AUTO_MIGRATE", "true"},
		{"非法端口", "STARS_DATABASE_PORT", "70000"},
		{"非法时间", "STARS_REDIS_TIMEOUT", "invalid"},
		{"负窗口", "STARS_RATE_LIMIT_WINDOW", "-1s"},
		{"非法节点", "STARS_ID_NODE", "1024"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := fixture(t, "dev")
			t.Setenv(test.key, test.value)
			c, err := Load(dir)
			if err == nil {
				err = c.ValidateServer()
			}
			if err == nil {
				t.Fatal("非法配置未被拒绝")
			}
		})
	}
}

func TestUnknownConfigurationKey(t *testing.T) {
	dir := fixture(t, "dev")
	if err := os.WriteFile(filepath.Join(dir, "config.dev.yaml"), []byte("app:\n  misspelled_timeout: 5s\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("拼写错误的配置项不应静默忽略")
	}
}

func TestMigrationIgnoresUnusedConfigurationTypes(t *testing.T) {
	dir := fixture(t, "prod")
	t.Setenv("STARS_DATABASE_PASSWORD", "migration-test-password")
	t.Setenv("STARS_REDIS_TIMEOUT", "invalid")
	t.Setenv("STARS_JWT_ACCESS_TTL", "invalid")
	t.Setenv("STARS_APP_READ_TIMEOUT", "invalid")
	cfg, err := LoadForMigration(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.ValidateDatabase(); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("HTTP 入口仍须拒绝非法配置类型")
	}
	t.Setenv("STARS_DATABASE_TIMEOUT", "invalid")
	if _, err := LoadForMigration(dir); err == nil {
		t.Fatal("迁移仍须拒绝非法数据库配置")
	}
}
