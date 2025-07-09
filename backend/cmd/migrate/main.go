package main

import (
	"log"
	"stars-admin/internal/config"
	"stars-admin/internal/database"
	"stars-admin/internal/models"
	"stars-admin/internal/utils"
	"time"

	"gorm.io/gorm"
)

// migrate 数据库迁移和初始化数据
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

	// 初始化数据
	if err := initData(db); err != nil {
		log.Fatal("Failed to initialize data:", err)
	}

	log.Println("Database migration completed successfully!")
}

// initData 初始化基础数据
func initData(db *gorm.DB) error {
	// 创建超级管理员用户
	if err := createSuperAdmin(db); err != nil {
		return err
	}

	// 创建基础角色
	if err := createBasicRoles(db); err != nil {
		return err
	}

	// 创建基础菜单
	if err := createBasicMenus(db); err != nil {
		return err
	}

	// 分配角色权限
	if err := assignRolePermissions(db); err != nil {
		return err
	}

	return nil
}

// createSuperAdmin 创建超级管理员用户
func createSuperAdmin(db *gorm.DB) error {
	var count int64
	db.Model(&models.User{}).Count(&count)
	
	if count == 0 {
		hashedPassword, err := utils.HashPassword("admin123")
		if err != nil {
			return err
		}

		admin := models.User{
			Username:  "admin",
			Password:  hashedPassword,
			Email:     "admin@example.com",
			Nickname:  "超级管理员",
			Status:    1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := db.Create(&admin).Error; err != nil {
			return err
		}

		log.Println("Created super admin user: admin/admin123")
	}

	return nil
}

// createBasicRoles 创建基础角色
func createBasicRoles(db *gorm.DB) error {
	roles := []models.Role{
		{
			Name:        "超级管理员",
			Code:        "admin",
			Description: "系统超级管理员，拥有所有权限",
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "普通用户",
			Code:        "user",
			Description: "普通用户，基础权限",
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, role := range roles {
		var count int64
		db.Model(&models.Role{}).Where("code = ?", role.Code).Count(&count)
		if count == 0 {
			if err := db.Create(&role).Error; err != nil {
				return err
			}
		}
	}

	// 给admin用户分配admin角色
	var adminUser models.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		return err
	}

	var adminRole models.Role
	if err := db.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}

	var userRoleCount int64
	db.Model(&models.UserRole{}).Where("user_id = ? AND role_id = ?", adminUser.ID, adminRole.ID).Count(&userRoleCount)
	if userRoleCount == 0 {
		userRole := models.UserRole{
			UserID:    adminUser.ID,
			RoleID:    adminRole.ID,
			CreatedAt: time.Now(),
		}
		if err := db.Create(&userRole).Error; err != nil {
			return err
		}
	}

	log.Println("Created basic roles")
	return nil
}

// createBasicMenus 创建基础菜单
func createBasicMenus(db *gorm.DB) error {
	menus := []models.Menu{
		{
			Name:      "系统管理",
			Path:      "/system",
			Icon:      "SettingOutlined",
			Sort:      1,
			Type:      1,
			Status:    1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:      "用户管理",
			Path:      "/system/users",
			Icon:      "UserOutlined",
			Sort:      1,
			Type:      1,
			Status:    1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:      "角色管理",
			Path:      "/system/roles",
			Icon:      "TeamOutlined",
			Sort:      2,
			Type:      1,
			Status:    1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:      "菜单管理",
			Path:      "/system/menus",
			Icon:      "MenuOutlined",
			Sort:      3,
			Type:      1,
			Status:    1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:      "操作日志",
			Path:      "/system/logs",
			Icon:      "FileTextOutlined",
			Sort:      4,
			Type:      1,
			Status:    1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, menu := range menus {
		var count int64
		db.Model(&models.Menu{}).Where("path = ?", menu.Path).Count(&count)
		if count == 0 {
			if err := db.Create(&menu).Error; err != nil {
				return err
			}
		}
	}

	// 设置父子关系
	var systemMenu models.Menu
	if err := db.Where("path = ?", "/system").First(&systemMenu).Error; err != nil {
		return err
	}

	childPaths := []string{"/system/users", "/system/roles", "/system/menus", "/system/logs"}
	for _, path := range childPaths {
		db.Model(&models.Menu{}).Where("path = ?", path).Update("parent_id", systemMenu.ID)
	}

	log.Println("Created basic menus")
	return nil
}

// assignRolePermissions 分配角色权限
func assignRolePermissions(db *gorm.DB) error {
	var adminRole models.Role
	if err := db.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}

	var menus []models.Menu
	if err := db.Find(&menus).Error; err != nil {
		return err
	}

	// 给admin角色分配所有菜单权限
	for _, menu := range menus {
		var count int64
		db.Model(&models.RoleMenu{}).Where("role_id = ? AND menu_id = ?", adminRole.ID, menu.ID).Count(&count)
		if count == 0 {
			roleMenu := models.RoleMenu{
				RoleID:    adminRole.ID,
				MenuID:    menu.ID,
				CreatedAt: time.Now(),
			}
			if err := db.Create(&roleMenu).Error; err != nil {
				return err
			}
		}
	}

	log.Println("Assigned role permissions")
	return nil
}