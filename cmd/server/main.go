// Package main 提供 HTTP 服务与数据库迁移入口。
package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/StarsSkyCat/Stars-Admin/internal/app"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/database"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// @title Stars-Admin API
// @version 0.1
// @description Stars-Admin 框架及 M2 认证与权限接口
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	gin.SetMode(gin.ReleaseMode)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		// CLI 的错误上下文属于内部诊断，不发送给 HTTP 客户端。
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) (err error) {
	if len(args) > 0 && args[0] != "serve" && args[0] != "migrate" {
		return errors.New("用法：stars-admin [serve | migrate up|down|status]")
	}
	migrating := len(args) > 0 && args[0] == "migrate"
	if migrating && (len(args) != 2 || (args[1] != "up" && args[1] != "down" && args[1] != "status")) {
		return errors.New("用法：stars-admin migrate up|down|status")
	}
	if !migrating && len(args) > 1 {
		return errors.New("serve 不接受额外参数")
	}
	dir := cmp.Or(os.Getenv("STARS_CONFIG_DIR"), "configs")
	var cfg *config.Config
	if migrating {
		cfg, err = config.LoadForMigration(dir)
	} else {
		cfg, err = config.Load(dir)
	}
	if err != nil {
		return err
	}
	if migrating {
		if err := cfg.ValidateDatabase(); err != nil {
			return err
		}
	} else {
		if err := cfg.ValidateServer(); err != nil {
			return err
		}
	}
	log, closeLog, err := logger.New(cfg.Log)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, closeLog()) }()
	password := os.Getenv("STARS_SEED_ADMIN_PASSWORD")
	if migrating {
		if args[1] != "status" {
			return database.Migrate(ctx, cfg, args[1], password)
		}
		connection, err := database.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer func() { _ = connection.Close() }()
		probe, cancel := context.WithTimeout(ctx, cfg.Database.Timeout)
		defer cancel()
		status, err := database.ReadStatus(probe, connection.SQL)
		if err == nil {
			// 状态是命令的标准输出数据，不是日志。
			_, err = fmt.Fprintf(os.Stdout, "version=%d expected=%d dirty=%t\n", status.Version, status.Expected, status.Dirty)
		}
		return err
	}
	if cfg.App.Env == "dev" {
		if err := database.Migrate(ctx, cfg, "up", password); err != nil {
			return err
		}
	}
	application, err := app.New(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, application.Close())
		log.Info("应用已停止", zap.Bool("success", err == nil))
	}()
	return application.Run(ctx)
}
