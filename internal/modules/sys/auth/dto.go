// Package auth 提供登录、令牌轮换和本人认证接口。
package auth

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/permission"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
)

// LoginInput 包含敏感字段，禁止序列化到日志。
type LoginInput struct {
	Username    string `json:"username" binding:"required,max=64"`
	Password    string `json:"password" binding:"required,max=72"`
	CaptchaUUID string `json:"captchaUuid"`
	CaptchaCode string `json:"captchaCode"`
}

// RefreshInput 只能接受 Refresh 令牌。
type RefreshInput struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=4096"`
}

// PasswordInput 修改本人密码必须同时提交旧密码。
type PasswordInput struct {
	OldPassword string `json:"oldPassword" binding:"required,max=72"`
	NewPassword string `json:"newPassword" binding:"required,max=72"`
}

// Captcha 返回可显示的图片，永不返回答案。
type Captcha struct {
	Enabled   bool   `json:"enabled"`
	UUID      string `json:"uuid"`
	Image     string `json:"image"`
	ExpiresIn int    `json:"expiresIn"`
}

// Profile 是当前身份、昵称、导航菜单及操作权限集合。
type Profile struct {
	security.Principal
	Nickname    string            `json:"nickname"`
	Permissions []string          `json:"permissions"`
	Menus       []permission.Menu `json:"menus"`
}
