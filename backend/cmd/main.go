package main

import (
	"log"
	"stars-admin/internal/config"
	"stars-admin/internal/database"
	"stars-admin/internal/api/routes"
	"stars-admin/internal/api/middleware"
	
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
)

// @title Stars Admin API
// @version 1.0
// @description Stars Admin后台管理系统API
// @host localhost:8080
// @BasePath /api/v1
func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 初始化数据库
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 初始化Redis
	rdb, err := database.InitRedis(cfg)
	if err != nil {
		log.Fatal("Failed to initialize Redis:", err)
	}

	// 设置Gin模式
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := gin.Default()

	// 添加CORS中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// 添加日志中间件
	r.Use(middleware.Logger())

	// 添加错误处理中间件
	r.Use(middleware.ErrorHandler())

	// 注册路由
	routes.RegisterRoutes(r, db, rdb)

	// 启动服务器
	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}