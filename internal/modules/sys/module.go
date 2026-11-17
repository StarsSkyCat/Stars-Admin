// Package sys 装配系统模块，业务功能从 M2 起实现。
package sys

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core"
	"github.com/gin-gonic/gin"
)

// Module 是 M1 的系统模块占位实现。
type Module struct{}

// New 构造模块，后续按功能注入所需依赖。
func New(_ core.Deps) *Module { return &Module{} }

// Name 返回模块标识。
func (*Module) Name() string { return "sys" }

// RegisterRoutes 在 M1 不注册业务路由。
func (*Module) RegisterRoutes(_ *gin.RouterGroup) {}
