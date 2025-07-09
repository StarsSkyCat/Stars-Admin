package routes

import (
	"stars-admin/internal/api/handlers"
	"stars-admin/internal/api/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// RegisterRoutes 注册路由
func RegisterRoutes(r *gin.Engine, db *gorm.DB, rdb *redis.Client) {
	// 创建处理器
	authHandler := handlers.NewAuthHandler(db, rdb)
	
	// API路由组
	api := r.Group("/api/v1")
	
	// 公共路由（不需要认证）
	public := api.Group("")
	{
		// 认证相关路由
		auth := public.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}
		
		// 健康检查
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
				"message": "Service is running",
			})
		})
	}
	
	// 私有路由（需要认证）
	private := api.Group("")
	private.Use(middleware.AuthMiddleware(db, rdb))
	private.Use(middleware.OperationLogger(db))
	{
		// 认证相关路由
		auth := private.Group("/auth")
		{
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/user", authHandler.GetUserInfo)
			auth.PUT("/password", authHandler.UpdatePassword)
		}
		
		// 用户管理路由
		users := private.Group("/users")
		{
			users.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "用户列表"})
			})
			users.POST("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "创建用户"})
			})
			users.GET("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "用户详情"})
			})
			users.PUT("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "更新用户"})
			})
			users.DELETE("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "删除用户"})
			})
		}
		
		// 角色管理路由
		roles := private.Group("/roles")
		{
			roles.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "角色列表"})
			})
			roles.POST("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "创建角色"})
			})
			roles.GET("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "角色详情"})
			})
			roles.PUT("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "更新角色"})
			})
			roles.DELETE("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "删除角色"})
			})
		}
		
		// 菜单管理路由
		menus := private.Group("/menus")
		{
			menus.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "菜单列表"})
			})
			menus.POST("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "创建菜单"})
			})
			menus.GET("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "菜单详情"})
			})
			menus.PUT("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "更新菜单"})
			})
			menus.DELETE("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "删除菜单"})
			})
		}
		
		// 系统管理路由
		system := private.Group("/system")
		{
			// 操作日志
			system.GET("/logs", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "操作日志"})
			})
			
			// 系统配置
			system.GET("/config", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "系统配置"})
			})
			system.PUT("/config", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "更新系统配置"})
			})
		}
	}
}