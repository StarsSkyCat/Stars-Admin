// Package app 手工装配基础设施、模块与生命周期。
package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/StarsSkyCat/Stars-Admin/internal/core"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/database"
	coreredis "github.com/StarsSkyCat/Stars-Admin/internal/core/redis"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/server"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/gen"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/job"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys"
	"github.com/StarsSkyCat/Stars-Admin/pkg/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// App 持有应用资源；模块后续通过生命周期钩子在依赖关闭前停止。
type App struct {
	cfg     *config.Config
	log     *zap.Logger
	db      *database.Connection
	cache   *redis.Client
	router  *gin.Engine
	ready   atomic.Bool
	cleanup server.Cleanup
	system  *sys.Module
}

// New 任意初始化步骤失败时按逆序释放已获得资源。
func New(ctx context.Context, cfg *config.Config, log *zap.Logger) (application *App, err error) {
	if err := cfg.ValidateServer(); err != nil {
		return nil, err
	}
	a := &App{cfg: cfg, log: log}
	defer func() {
		if err != nil {
			err = errors.Join(err, a.cleanup.Close())
		}
	}()
	a.db, err = database.Open(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	a.cleanup.Add(a.db.Close)
	a.cache, err = coreredis.Open(ctx, cfg.Redis)
	if err != nil {
		return nil, err
	}
	a.cleanup.Add(a.cache.Close)
	check, cancel := context.WithTimeout(ctx, cfg.Database.Timeout)
	defer cancel()
	if err = database.CheckMigrations(check, a.db.SQL); err != nil {
		return nil, err
	}
	ids, err := snowflake.New(cfg.ID.Node)
	if err != nil {
		return nil, err
	}
	deps := core.Deps{Config: cfg, DB: a.db.ORM, Redis: a.cache, Logger: log, IDs: ids}
	a.system, err = sys.New(ctx, deps)
	if err != nil {
		return nil, err
	}
	a.cleanup.Add(a.system.Close)
	a.router, err = NewRouter(cfg, log, a.cache, a.Ready, a.system, job.New(deps), gen.New(deps))
	if err != nil {
		return nil, err
	}
	a.ready.Store(true)
	return a, nil
}

// Ready 校验生命周期状态、数据库版本与 Redis，不向客户端暴露具体失败原因。
func (a *App) Ready(ctx context.Context) error {
	if !a.ready.Load() {
		return errors.New("应用正在停止或尚未初始化")
	}
	databaseCtx, cancelDB := context.WithTimeout(ctx, a.cfg.Database.Timeout)
	defer cancelDB()
	if err := database.CheckMigrations(databaseCtx, a.db.SQL); err != nil {
		return err
	}
	if err := a.system.Ready(ctx); err != nil {
		return err
	}
	redisCtx, cancelRedis := context.WithTimeout(ctx, a.cfg.Redis.Timeout)
	defer cancelRedis()
	return a.cache.Ping(redisCtx).Err()
}

// Run 在上下文取消时先撤下就绪状态，再等待 HTTP 完成。
func (a *App) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", a.cfg.App.Addr)
	if err != nil {
		return fmt.Errorf("监听 HTTP 地址失败: %w", err)
	}
	a.log.Info("HTTP 服务已启动", zap.String("addr", listener.Addr().String()), zap.String("env", a.cfg.App.Env))
	return server.Run(ctx, listener, a.router, a.cfg.App, func() {
		a.ready.Store(false)
		a.log.Info("HTTP 服务开始优雅停机")
	})
}

// Close 必须在 HTTP 与模块后台工作停止后调用。
func (a *App) Close() error {
	a.ready.Store(false)
	return a.cleanup.Close()
}
