package services

import (
	"errors"
	"stars-admin/internal/models"
	"stars-admin/internal/utils"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// AuthService 认证服务
type AuthService struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewAuthService 创建认证服务
func NewAuthService(db *gorm.DB, rdb *redis.Client) *AuthService {
	return &AuthService{
		db:  db,
		rdb: rdb,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"`
	User         *UserInfo `json:"user"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Status   int    `json:"status"`
}

// RefreshTokenRequest 刷新token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login 用户登录
func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	// 查找用户
	var user models.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, err
	}

	// 检查用户状态
	if user.Status != 1 {
		return nil, errors.New("用户已被禁用")
	}

	// 验证密码
	if !utils.CheckPassword(user.Password, req.Password) {
		return nil, errors.New("用户名或密码错误")
	}

	// 获取用户角色和权限
	roles, permissions, err := s.getUserRolesAndPermissions(user.ID)
	if err != nil {
		return nil, err
	}

	// 生成JWT token
	accessToken, err := utils.GenerateJWT(user.ID, user.Username, roles, permissions)
	if err != nil {
		return nil, err
	}

	// 生成刷新token
	refreshToken := utils.GenerateRefreshToken()
	if err := utils.StoreRefreshToken(s.rdb, user.ID, refreshToken, 7*24*time.Hour); err != nil {
		return nil, err
	}

	// 更新最后登录时间
	now := time.Now()
	s.db.Model(&user).Update("last_login_at", &now)

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    24 * 60 * 60, // 24小时
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			Status:   user.Status,
		},
	}, nil
}

// RefreshToken 刷新访问令牌
func (s *AuthService) RefreshToken(req *RefreshTokenRequest) (*LoginResponse, error) {
	// 这里需要从刷新token中获取用户信息
	// 简化实现，实际应该存储刷新token与用户的关联
	return nil, errors.New("refresh token功能待实现")
}

// Logout 用户登出
func (s *AuthService) Logout(userID uint, token string) error {
	// 将token加入黑名单
	if err := utils.BlacklistToken(s.rdb, token, 24*time.Hour); err != nil {
		return err
	}

	// 删除刷新token
	if err := utils.DeleteRefreshToken(s.rdb, userID); err != nil {
		return err
	}

	return nil
}

// GetUserInfo 获取用户信息
func (s *AuthService) GetUserInfo(userID uint) (*UserInfo, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	return &UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Status:   user.Status,
	}, nil
}

// UpdatePassword 更新密码
func (s *AuthService) UpdatePassword(userID uint, oldPassword, newPassword string) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	// 验证旧密码
	if !utils.CheckPassword(user.Password, oldPassword) {
		return errors.New("旧密码错误")
	}

	// 加密新密码
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// 更新密码
	return s.db.Model(&user).Update("password", hashedPassword).Error
}

// getUserRolesAndPermissions 获取用户角色和权限
func (s *AuthService) getUserRolesAndPermissions(userID uint) ([]string, []string, error) {
	var roles []models.Role
	if err := s.db.Joins("JOIN xc_user_roles ON xc_user_roles.role_id = xc_roles.id").
		Where("xc_user_roles.user_id = ? AND xc_roles.status = 1", userID).
		Find(&roles).Error; err != nil {
		return nil, nil, err
	}

	var roleNames []string
	var permissions []string

	for _, role := range roles {
		roleNames = append(roleNames, role.Code)

		// 获取角色对应的菜单权限
		var menus []models.Menu
		if err := s.db.Joins("JOIN xc_role_menus ON xc_role_menus.menu_id = xc_menus.id").
			Where("xc_role_menus.role_id = ? AND xc_menus.status = 1", role.ID).
			Find(&menus).Error; err != nil {
			continue
		}

		for _, menu := range menus {
			if menu.Path != "" {
				permissions = append(permissions, menu.Path)
			}
		}
	}

	return roleNames, permissions, nil
}
