// Package job 装配单实例任务配置、调度与异步日志生命周期。
package job

import (
	"context"
	"errors"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/scheduler"
	joblog "github.com/StarsSkyCat/Stars-Admin/internal/modules/job/log"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/job/task"
	"github.com/gin-gonic/gin"
)

// Module 在基础设施关闭前等待任务和日志。
type Module struct {
	tasks      *task.Handler
	logs       *joblog.Handler
	service    *task.Service
	engine     *scheduler.Engine
	logService *joblog.Service
	guards     httpx.Guards
	timeout    time.Duration
}

// New 只装配已注册的可信任务目标，不导入 sys。
func New(ctx context.Context, deps core.Deps, guards httpx.Guards, registry *scheduler.Registry) *Module {
	logs := joblog.New(deps.DB, deps.IDs, deps.Logger)
	engine := scheduler.New(registry, logs, deps.Config.Job.Timeout)
	tasks := task.New(ctx, deps.DB, deps.IDs, engine, deps.Logger, deps.Config.Database.Timeout)
	return &Module{task.NewHandler(tasks), joblog.NewHandler(logs), tasks, engine, logs, guards, deps.Config.Database.Timeout}
}

// Name 返回模块标识。
func (*Module) Name() string { return "job" }

// RegisterRoutes 安全能力由 app 注入，模块不依赖认证业务实现。
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Use(httpx.Deadline(m.timeout))
	m.tasks.RegisterRoutes(rg.Group("/tasks"), m.guards)
	m.logs.RegisterRoutes(rg.Group("/logs"), m.guards)
}

// Ready 暴露配置与内存调度计划是否同步。
func (m *Module) Ready(ctx context.Context) error { return m.service.Ready(ctx) }

// StopScheduling 先关闭所有新触发入口并取消在途任务 context。
func (m *Module) StopScheduling() { m.engine.StopScheduling(); m.service.Stop() }

// Close 必须在数据库之前关闭，先等任务再排空日志。
func (m *Module) Close() error {
	m.StopScheduling()
	return errors.Join(m.engine.Close(), m.logService.Close())
}
