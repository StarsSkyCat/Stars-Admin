// Package dept 提供部门树管理与事务内祖级路径维护。
package dept

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
)

// Input 仅包含允许编辑的部门业务字段。
type Input struct {
	ParentID model.ID `json:"parentId"`
	Name     string   `json:"name" binding:"required,max=64"`
	Sort     int      `json:"sort"`
	Leader   *string  `json:"leader" binding:"omitempty,max=64"`
	Phone    *string  `json:"phone" binding:"omitempty,max=20"`
	Status   int      `json:"status" binding:"required,oneof=1 2"`
	Remark   *string  `json:"remark" binding:"omitempty,max=500"`
}

// View 是部门公开结构，根节点 parentId 为字符串零。
type View struct {
	httpx.Meta
	Input
	Ancestors string `json:"ancestors"`
	Children  []View `json:"children"`
}

// Query 是白名单过滤参数。
type Query struct {
	response.PageQuery
	Name   string `form:"name"`
	Status int    `form:"status"`
}

func view(d department) View {
	return View{ID: d.ID, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		Input: Input{ParentID: d.ParentID, Name: d.Name, Sort: d.Sort, Leader: d.Leader, Phone: d.Phone, Status: d.Status, Remark: d.Remark}, Ancestors: d.Ancestors, Children: []View{}}
}
