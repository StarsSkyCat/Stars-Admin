// Package user 提供用户管理及对外的认证能力接口。
package user

import (
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
)

// Input 资料编辑不接受密码、状态、授权或审计字段。
type Input struct {
	Username string    `json:"username" binding:"required,max=64"`
	Nickname string    `json:"nickname" binding:"required,max=64"`
	Email    *string   `json:"email" binding:"omitempty,max=128"`
	Phone    *string   `json:"phone" binding:"omitempty,max=20"`
	Gender   int       `json:"gender" binding:"oneof=0 1 2"`
	Avatar   *string   `json:"avatar" binding:"omitempty,max=255"`
	DeptID   *model.ID `json:"deptId"`
	Remark   *string   `json:"remark" binding:"omitempty,max=500"`
}

// CreateInput 只有创建接口可以接受初始密码。
type CreateInput struct {
	Input
	Password string `json:"password" binding:"required"`
}

// View 永不返回密码哈希和内部会话版本。
type View struct {
	httpx.Meta
	Input
	Status       int        `json:"status"`
	LoginIP      *string    `json:"loginIp"`
	LoginAt      *time.Time `json:"loginAt"`
	PwdUpdatedAt *time.Time `json:"pwdUpdatedAt"`
	RoleIDs      []model.ID `json:"roleIds"`
	PostIDs      []model.ID `json:"postIds"`
}

// Query 提供明确的过滤字段。
type Query struct {
	response.PageQuery
	Username string    `form:"username"`
	Nickname string    `form:"nickname"`
	Status   int       `form:"status"`
	DeptID   *model.ID `form:"deptId"`
}

// StatusInput 由状态入口触发安全撤销。
type StatusInput struct {
	Status int `json:"status" binding:"required,oneof=1 2"`
}

// PasswordInput 用于管理员重置，初始密码仍须满足长度边界。
type PasswordInput struct {
	Password string `json:"password" binding:"required"`
}

// RolesInput 只接受显式角色 ID 列表。
type RolesInput struct {
	RoleIDs []model.ID `json:"roleIds" binding:"required"`
}

func view(u user) View {
	return View{ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Input: Input{Username: u.Username, Nickname: u.Nickname, Email: u.Email, Phone: u.Phone, Gender: u.Gender, Avatar: u.Avatar, DeptID: u.DeptID, Remark: u.Remark}, Status: u.Status, LoginIP: u.LoginIP, LoginAt: u.LoginAt, PwdUpdatedAt: u.PwdUpdatedAt, RoleIDs: []model.ID{}, PostIDs: []model.ID{}}
}

// PostsInput 是用户岗位绑定参数。
type PostsInput struct {
	PostIDs []model.ID `json:"postIds" binding:"required"`
}
