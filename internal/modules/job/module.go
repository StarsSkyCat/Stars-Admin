// Package job 装配定时任务模块，调度功能在 M4 实现。
package job

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core"
	"github.com/gin-gonic/gin"
)

// Module 是 M1 的任务模块占位实现。
type Module struct{}

// New 构造模块。
func New(_ core.Deps) *Module { return &Module{} }

// Name 返回模块标识。
func (*Module) Name() string { return "job" }

// RegisterRoutes 在 M1 不启动调度器或注册业务路由。
func (*Module) RegisterRoutes(_ *gin.RouterGroup) {}
