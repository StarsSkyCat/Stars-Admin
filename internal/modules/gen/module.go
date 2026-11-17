// Package gen 装配受限数据库元数据与固定模板代码生成。
package gen

import (
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/project"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/gen/table"
	"github.com/gin-gonic/gin"
)

// Module 不执行生成 SQL，也不会自动装载生成代码。
type Module struct {
	tables  *table.Handler
	guards  httpx.Guards
	timeout time.Duration
}

// New 从实际 go.mod 或构建信息读取 import 根路径。
func New(deps core.Deps, guards httpx.Guards) (*Module, error) {
	module, err := project.ModulePath()
	if err != nil {
		return nil, err
	}
	service := table.New(deps.DB, deps.IDs, deps.Config.Gen, deps.Config.App.Env, module)
	return &Module{table.NewHandler(service), guards, deps.Config.Database.Timeout}, nil
}

// Name 返回模块标识。
func (*Module) Name() string { return "gen" }

// RegisterRoutes 所有生成管理默认使用仅 admin 的逐动作授权。
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Use(httpx.Deadline(m.timeout))
	m.tables.RegisterRoutes(rg.Group("/tables"), m.guards)
}
