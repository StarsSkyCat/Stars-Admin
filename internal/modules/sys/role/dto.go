// Package role 提供角色与专用授权操作。
package role

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
)

// Input 普通编辑不能改状态、数据范围或授权关系。
type Input struct {
	Name   string  `json:"name" binding:"required,max=64"`
	Code   string  `json:"code" binding:"required,max=64"`
	Sort   int     `json:"sort"`
	Remark *string `json:"remark" binding:"omitempty,max=500"`
}

// View 只公开业务字段和授权 ID。
type View struct {
	httpx.Meta
	Input
	Status    int        `json:"status"`
	DataScope int        `json:"dataScope"`
	MenuIDs   []model.ID `json:"menuIds"`
	DeptIDs   []model.ID `json:"deptIds"`
}

// Query 定义角色列表过滤。
type Query struct {
	response.PageQuery
	Name   string `form:"name"`
	Code   string `form:"code"`
	Status int    `form:"status"`
}

// MenuInput 显式替换当前菜单授权。
type MenuInput struct {
	MenuIDs []model.ID `json:"menuIds" binding:"required"`
}

// ScopeInput 五档设置在 M2 存储，数据过滤在 M3 实现。
type ScopeInput struct {
	DataScope int        `json:"dataScope" binding:"required,min=1,max=5"`
	DeptIDs   []model.ID `json:"deptIds"`
}

// StatusInput 通过专用路由触发会话撤销。
type StatusInput struct {
	Status int `json:"status" binding:"required,oneof=1 2"`
}

func view(r role) View {
	return View{ID: r.ID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Input: Input{Name: r.Name, Code: r.Code, Sort: r.Sort, Remark: r.Remark}, Status: r.Status, DataScope: r.DataScope, MenuIDs: []model.ID{}, DeptIDs: []model.ID{}}
}
