// Package menu 管理菜单、按钮及路由权限标识。
package menu

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
)

// Input 不包含 ID、审计人或客户端可写的授权关系。
type Input struct {
	ParentID  model.ID `json:"parentId"`
	Name      string   `json:"name" binding:"required,max=64"`
	Type      int      `json:"type" binding:"required,oneof=1 2 3"`
	Path      *string  `json:"path" binding:"omitempty,max=255"`
	Component *string  `json:"component" binding:"omitempty,max=255"`
	Perm      *string  `json:"perm" binding:"omitempty,max=128"`
	Icon      *string  `json:"icon" binding:"omitempty,max=64"`
	Sort      int      `json:"sort"`
	Visible   int      `json:"visible" binding:"required,oneof=1 2"`
	Status    int      `json:"status" binding:"required,oneof=1 2"`
	Remark    *string  `json:"remark" binding:"omitempty,max=500"`
}

// View 是菜单响应；隐藏不等于停用，隐藏节点仍可能提供按钮权限。
type View struct {
	httpx.Meta
	Input
	Children []View `json:"children"`
}

// Query 提供名称与状态过滤。
type Query struct {
	response.PageQuery
	Name   string `form:"name"`
	Status int    `form:"status"`
}

func view(m menu) View {
	return View{ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		Input: Input{ParentID: m.ParentID, Name: m.Name, Type: m.Type, Path: m.Path, Component: m.Component, Perm: m.Perm, Icon: m.Icon, Sort: m.Sort, Visible: m.Visible, Status: m.Status, Remark: m.Remark}, Children: []View{}}
}
