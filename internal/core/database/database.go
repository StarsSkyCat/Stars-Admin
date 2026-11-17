// Package database 集中 PostgreSQL 驱动、连接池及版本化迁移。
package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"
)

// Connection 同时提供 GORM 与底层连接池，不交给 HTTP handler。
type Connection struct {
	ORM *gorm.DB
	SQL *sql.DB
}

// DSN 正确转义账号密码，禁止将返回值写入日志。
func DSN(c config.Database) string {
	u := &url.URL{Scheme: "postgres", User: url.UserPassword(c.Username, c.Password), Host: net.JoinHostPort(c.Host, strconv.Itoa(c.Port)), Path: "/" + c.DBName}
	values := u.Query()
	values.Set("sslmode", c.SSLMode)
	values.Set("TimeZone", "UTC")
	values.Set("application_name", "stars-admin")
	values.Set("connect_timeout", strconv.Itoa(max(1, int(c.Timeout.Seconds()))))
	u.RawQuery = values.Encode()
	return u.String()
}

// Open 建立连接并检查连通性，失败时关闭已申请资源。
func Open(ctx context.Context, c config.Database) (*Connection, error) {
	db, err := gorm.Open(postgres.Open(DSN(c)), &gorm.Config{
		DisableAutomaticPing: true,
		TranslateError:       true,
		Logger:               gormlog.Discard,
		NowFunc:              func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 PostgreSQL 失败: %w", err)
	}
	pool, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %w", err)
	}
	pool.SetMaxOpenConns(c.MaxOpenConns)
	pool.SetMaxIdleConns(c.MaxIdleConns)
	pool.SetConnMaxLifetime(c.ConnMaxLifetime)
	probe, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	if err := pool.PingContext(probe); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("PostgreSQL 连通检查失败: %w", err)
	}
	return &Connection{ORM: db, SQL: pool}, nil
}

// Close 关闭底层池，关闭后不得继续向数据库写入。
func (c *Connection) Close() error { return c.SQL.Close() }
