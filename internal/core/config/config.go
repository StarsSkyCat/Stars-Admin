// Package config 负责配置合并、环境变量绑定及按命令校验。
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 是应用配置，敏感配置禁止整体序列化或写入日志。
type Config struct {
	App       App       `mapstructure:"app"`
	Database  Database  `mapstructure:"database"`
	Redis     Redis     `mapstructure:"redis"`
	ID        ID        `mapstructure:"id"`
	JWT       JWT       `mapstructure:"jwt"`
	Auth      Auth      `mapstructure:"auth"`
	Log       Log       `mapstructure:"log"`
	Swagger   Swagger   `mapstructure:"swagger"`
	CORS      CORS      `mapstructure:"cors"`
	RateLimit RateLimit `mapstructure:"rate_limit"`
}

// App 定义 HTTP 服务和停机参数。
type App struct {
	Name              string        `mapstructure:"name"`
	Env               string        `mapstructure:"env"`
	Addr              string        `mapstructure:"addr"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
	TrustedProxies    []string      `mapstructure:"trusted_proxies"`
}

// Database 是当前唯一实现的 PostgreSQL 连接配置。
type Database struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	Timeout         time.Duration `mapstructure:"timeout"`
	AutoMigrate     bool          `mapstructure:"auto_migrate"`
}

// Redis 定义 Redis 客户端和键前缀。
type Redis struct {
	Addr      string        `mapstructure:"addr"`
	Password  string        `mapstructure:"password"`
	DB        int           `mapstructure:"db"`
	PoolSize  int           `mapstructure:"pool_size"`
	KeyPrefix string        `mapstructure:"key_prefix"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

// ID 定义由部署分配的雪花节点编号。
type ID struct {
	Node int64 `mapstructure:"node"`
}

// JWT 定义限定用途与会话绝对期限的双令牌配置。
type JWT struct {
	Secret     string        `mapstructure:"secret"`
	AccessTTL  time.Duration `mapstructure:"access_ttl"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
	Issuer     string        `mapstructure:"issuer"`
}

// Auth 定义进程级密码哈希成本，业务开关由 sys_config 只读服务管理。
type Auth struct {
	BcryptCost int `mapstructure:"bcrypt_cost"`
}

// Log 定义结构化日志和文件轮转参数。
type Log struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// Swagger 控制在线文档是否挂载。
type Swagger struct {
	Enabled bool `mapstructure:"enabled"`
}

// CORS 定义跨域来源，不启用跨域凭据。
type CORS struct {
	Origins []string `mapstructure:"origins"`
}

// RateLimit 定义按客户端 IP 与路由使用的固定窗口。
type RateLimit struct {
	Enabled  bool          `mapstructure:"enabled"`
	Requests int           `mapstructure:"requests"`
	Window   time.Duration `mapstructure:"window"`
}

// Load 合并基础、环境文件和显式绑定的环境变量，不在此连接任何服务。
func Load(dir string) (*Config, error) {
	return load(dir, false)
}

// LoadForMigration 只解码数据库与日志，损坏的 HTTP/Redis/JWT 配置不能阻碍结构修复。
func LoadForMigration(dir string) (*Config, error) {
	return load(dir, true)
}

// LoadForSeed 只解码 seed 所需字段，保持 CLI 与 HTTP/Redis/JWT 解耦。
func LoadForSeed(dir string) (*Config, error) {
	return loadCommand(dir, []string{"database", "log", "auth", "id"})
}

func load(dir string, migrationOnly bool) (*Config, error) {
	var sections []string
	if migrationOnly {
		sections = []string{"database", "log"}
	}
	return loadCommand(dir, sections)
}

func loadCommand(dir string, sections []string) (*Config, error) {
	env, exists := os.LookupEnv("APP_ENV")
	if !exists {
		env = "dev"
	}
	if !slices.Contains([]string{"dev", "test", "prod"}, env) {
		return nil, errors.New("APP_ENV 必须是 dev、test 或 prod")
	}
	v := viper.New()
	v.SetConfigFile(filepath.Join(dir, "config.yaml"))
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取基础配置失败: %w", err)
	}
	path := filepath.Join(dir, "config."+env+".yaml")
	f, err := os.Open(path)
	if err == nil {
		defer func() { _ = f.Close() }()
		if err := v.MergeConfig(f); err != nil {
			return nil, fmt.Errorf("合并环境配置失败: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) || env == "prod" {
		return nil, fmt.Errorf("读取环境配置失败: %w", err)
	}
	v.SetEnvPrefix("STARS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AllowEmptyEnv(true)
	if err := bind(v, reflect.TypeFor[Config](), ""); err != nil {
		return nil, err
	}
	var c Config
	decoder := v
	if sections != nil {
		// AllSettings 已合并显式环境变量；筛选后再解码，避免校验未使用字段的类型。
		settings := v.AllSettings()
		decoder = viper.New()
		for _, section := range sections {
			decoder.Set(section, settings[section])
		}
	}
	if err := decoder.UnmarshalExact(&c); err != nil {
		return nil, fmt.Errorf("配置类型或字段错误: %w", err)
	}
	// APP_ENV 是环境选择的唯一来源，不能由 YAML 或 STARS_APP_ENV 改写。
	c.App.Env = env
	return &c, nil
}

// ValidateSeed 仅验证初始化所用的数据库、哈希成本与 ID 节点。
func (c *Config) ValidateSeed() error {
	if err := c.ValidateDatabase(); err != nil {
		return err
	}
	if c.Auth.BcryptCost < 4 || c.Auth.BcryptCost > 14 || (c.App.Env == "prod" && c.Auth.BcryptCost < 10) || c.ID.Node < 0 || c.ID.Node > 1023 {
		return errors.New("seed 哈希成本或雪花节点配置无效")
	}
	return nil
}

func bind(v *viper.Viper, typ reflect.Type, prefix string) error {
	for i := range typ.NumField() {
		field := typ.Field(i)
		key := field.Tag.Get("mapstructure")
		if prefix != "" {
			key = prefix + "." + key
		}
		if field.Type.Kind() == reflect.Struct {
			if err := bind(v, field.Type, key); err != nil {
				return err
			}
		} else if err := v.BindEnv(key); err != nil {
			return fmt.Errorf("绑定环境变量失败: %w", err)
		}
	}
	return nil
}

// ValidateDatabase 仅校验迁移和数据库初始化需要的参数。
func (c *Config) ValidateDatabase() error {
	d := c.Database
	if d.Driver != "postgres" || d.Host == "" || d.Username == "" || d.DBName == "" || d.Port < 1 || d.Port > 65535 {
		return errors.New("数据库驱动、地址、端口、账号或库名无效")
	}
	if !slices.Contains([]string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}, d.SSLMode) {
		return errors.New("数据库 sslmode 无效")
	}
	if d.MaxOpenConns < 1 || d.MaxIdleConns < 0 || d.MaxIdleConns > d.MaxOpenConns || d.Timeout <= 0 || d.ConnMaxLifetime <= 0 {
		return errors.New("数据库连接池或超时配置无效")
	}
	if d.AutoMigrate {
		return errors.New("当前工程仅允许版本化 SQL 迁移，AutoMigrate 必须关闭")
	}
	if c.App.Env == "prod" {
		value, ok := os.LookupEnv("STARS_DATABASE_PASSWORD")
		if !ok || value == "" || value == "stars123" || value != d.Password {
			return errors.New("生产数据库密码必须通过环境变量提供且不能使用开发默认值")
		}
	}
	return nil
}

// ValidateServer 在启动 HTTP 前校验全部依赖与生产约束。
func (c *Config) ValidateServer() error {
	if err := c.ValidateDatabase(); err != nil {
		return err
	}
	if c.App.Name == "" || c.App.Addr == "" || c.App.ReadHeaderTimeout <= 0 || c.App.ReadTimeout <= 0 || c.App.WriteTimeout <= 0 || c.App.IdleTimeout <= 0 || c.App.ShutdownTimeout <= 0 {
		return errors.New("HTTP 地址、名称或超时配置无效")
	}
	if c.Redis.Addr == "" || c.Redis.DB < 0 || c.Redis.PoolSize < 1 || c.Redis.Timeout <= 0 || !strings.HasSuffix(c.Redis.KeyPrefix, ":") {
		return errors.New("Redis 地址、连接池、超时或键前缀无效")
	}
	if c.ID.Node < 0 || c.ID.Node > 1023 {
		return errors.New("雪花节点编号必须在 0 到 1023 之间")
	}
	if c.RateLimit.Requests < 1 || c.RateLimit.Window < time.Millisecond {
		return errors.New("限流次数必须为正数，窗口不能小于一毫秒")
	}
	if c.JWT.AccessTTL <= 0 || c.JWT.RefreshTTL <= c.JWT.AccessTTL || c.JWT.Issuer == "" {
		return errors.New("令牌有效期或签发方配置无效")
	}
	if c.JWT.Secret == "" || c.Auth.BcryptCost < 4 || c.Auth.BcryptCost > 14 || (c.App.Env == "prod" && c.Auth.BcryptCost < 10) {
		return errors.New("JWT 密钥或 bcrypt 成本配置无效，生产成本至少为 10")
	}
	if c.Log.Format != "json" && c.Log.Format != "console" {
		return errors.New("日志格式必须为 json 或 console")
	}
	if c.Log.MaxSize < 1 || c.Log.MaxBackups < 0 || c.Log.MaxAge < 0 {
		return errors.New("日志轮转配置无效")
	}
	if c.App.Env == "prod" {
		secret, ok := os.LookupEnv("STARS_JWT_SECRET")
		if !ok || len(secret) < 32 || secret == "dev-secret-change-me" || secret != c.JWT.Secret || c.Log.Format != "json" {
			return errors.New("生产环境必须通过环境变量提供至少 32 字节 JWT 密钥并使用 JSON 日志")
		}
		if slices.Contains(c.CORS.Origins, "*") {
			return errors.New("生产 CORS 必须使用明确的来源白名单")
		}
	}
	return nil
}
