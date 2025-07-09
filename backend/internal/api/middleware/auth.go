package middleware

import (
	"net/http"
	"strings"
	"stars-admin/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization header is required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid authorization format",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 提取token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		
		// 验证token
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 检查token是否在黑名单中
		if utils.IsTokenBlacklisted(rdb, tokenString) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Token is blacklisted",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user", claims)
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		
		c.Next()
	}
}

// RequirePermission 权限验证中间件
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "User not authenticated",
				"data":    nil,
			})
			c.Abort()
			return
		}

		claims, ok := user.(*utils.JWTClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid user claims",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 检查用户权限
		if !hasPermission(claims.Permissions, permission) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Permission denied",
				"data":    nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasPermission 检查用户是否有指定权限
func hasPermission(userPermissions []string, required string) bool {
	for _, permission := range userPermissions {
		if permission == required || permission == "*" {
			return true
		}
	}
	return false
}

// RequireRole 角色验证中间件
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "User not authenticated",
				"data":    nil,
			})
			c.Abort()
			return
		}

		claims, ok := user.(*utils.JWTClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid user claims",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 检查用户角色
		if !hasRole(claims.Roles, role) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Role denied",
				"data":    nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasRole 检查用户是否有指定角色
func hasRole(userRoles []string, required string) bool {
	for _, role := range userRoles {
		if role == required || role == "admin" {
			return true
		}
	}
	return false
}