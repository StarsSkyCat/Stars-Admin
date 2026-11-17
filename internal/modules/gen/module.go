// Package gen 装配代码生成模块，生成能力在 M4 实现。
package gen

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core"
	"github.com/gin-gonic/gin"
)

// Module 是 M1 的生成模块占位实现。
type Module struct{}

// New 构造模块。
func New(_ core.Deps) *Module { return &Module{} }

// Name 返回模块标识。
func (*Module) Name() string { return "gen" }

// RegisterRoutes 在 M1 不注册业务路由。
func (*Module) RegisterRoutes(_ *gin.RouterGroup) {}
