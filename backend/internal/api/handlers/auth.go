package handlers

import (
	"strings"
	"stars-admin/internal/services"
	"stars-admin/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(db *gorm.DB, rdb *redis.Client) *AuthHandler {
	return &AuthHandler{
		authService: services.NewAuthService(db, rdb),
	}
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body services.LoginRequest true "登录信息"
// @Success 200 {object} utils.Response{data=services.LoginResponse}
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidateError(c, err)
		return
	}

	resp, err := h.authService.Login(&req)
	if err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, resp)
}

// RefreshToken 刷新访问令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body services.RefreshTokenRequest true "刷新令牌"
// @Success 200 {object} utils.Response{data=services.LoginResponse}
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req services.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidateError(c, err)
		return
	}

	resp, err := h.authService.RefreshToken(&req)
	if err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, resp)
}

// Logout 用户登出
// @Summary 用户登出
// @Description 用户登出并注销访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerToken
// @Success 200 {object} utils.Response
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "用户未登录")
		return
	}

	// 获取token
	authHeader := c.GetHeader("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")

	if err := h.authService.Logout(userID.(uint), token); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, nil)
}

// GetUserInfo 获取用户信息
// @Summary 获取用户信息
// @Description 获取当前登录用户的信息
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerToken
// @Success 200 {object} utils.Response{data=services.UserInfo}
// @Router /auth/user [get]
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "用户未登录")
		return
	}

	userInfo, err := h.authService.GetUserInfo(userID.(uint))
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, userInfo)
}

// UpdatePassword 更新密码
// @Summary 更新密码
// @Description 更新当前用户的密码
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerToken
// @Param request body UpdatePasswordRequest true "密码信息"
// @Success 200 {object} utils.Response
// @Router /auth/password [put]
func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "用户未登录")
		return
	}

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidateError(c, err)
		return
	}

	if err := h.authService.UpdatePassword(userID.(uint), req.OldPassword, req.NewPassword); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "密码更新成功", nil)
}

// UpdatePasswordRequest 更新密码请求
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}