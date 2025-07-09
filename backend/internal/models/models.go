package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Username    string         `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password    string         `gorm:"size:255;not null" json:"-"`
	Email       string         `gorm:"uniqueIndex;size:100" json:"email"`
	Phone       string         `gorm:"size:20" json:"phone"`
	Nickname    string         `gorm:"size:50" json:"nickname"`
	Avatar      string         `gorm:"size:255" json:"avatar"`
	Status      int            `gorm:"default:1" json:"status"` // 1:正常 0:禁用
	LastLoginAt *time.Time     `json:"last_login_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Roles []Role `gorm:"many2many:xc_user_roles" json:"roles,omitempty"`
}

// Role 角色模型
type Role struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"uniqueIndex;size:50;not null" json:"name"`
	Code        string         `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Description string         `gorm:"size:255" json:"description"`
	Status      int            `gorm:"default:1" json:"status"` // 1:正常 0:禁用
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Users []User `gorm:"many2many:xc_user_roles" json:"users,omitempty"`
	Menus []Menu `gorm:"many2many:xc_role_menus" json:"menus,omitempty"`
}

// Menu 菜单模型
type Menu struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ParentID  uint           `gorm:"default:0" json:"parent_id"`
	Name      string         `gorm:"size:50;not null" json:"name"`
	Path      string         `gorm:"size:255" json:"path"`
	Component string         `gorm:"size:255" json:"component"`
	Icon      string         `gorm:"size:50" json:"icon"`
	Sort      int            `gorm:"default:0" json:"sort"`
	Type      int            `gorm:"default:1" json:"type"`   // 1:菜单 2:按钮
	Status    int            `gorm:"default:1" json:"status"` // 1:正常 0:禁用
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Children []Menu `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Roles    []Role `gorm:"many2many:xc_role_menus" json:"roles,omitempty"`
}

// Permission 权限模型
type Permission struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:50;not null" json:"name"`
	Code        string         `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Description string         `gorm:"size:255" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// UserRole 用户角色关联模型
type UserRole struct {
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	RoleID    uint      `gorm:"primaryKey" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

// RoleMenu 角色菜单关联模型
type RoleMenu struct {
	RoleID    uint      `gorm:"primaryKey" json:"role_id"`
	MenuID    uint      `gorm:"primaryKey" json:"menu_id"`
	CreatedAt time.Time `json:"created_at"`
}

// OperationLog 操作日志模型
type OperationLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	Username  string    `gorm:"size:50" json:"username"`
	Method    string    `gorm:"size:10" json:"method"`
	Path      string    `gorm:"size:255" json:"path"`
	IP        string    `gorm:"size:50" json:"ip"`
	UserAgent string    `gorm:"size:255" json:"user_agent"`
	Status    int       `json:"status"`
	Latency   int64     `json:"latency"` // 响应时间(毫秒)
	Request   string    `gorm:"type:text" json:"request"`
	Response  string    `gorm:"type:text" json:"response"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 设置表名
func (User) TableName() string {
	return "xc_users"
}

func (Role) TableName() string {
	return "xc_roles"
}

func (Menu) TableName() string {
	return "xc_menus"
}

func (Permission) TableName() string {
	return "xc_permissions"
}

func (UserRole) TableName() string {
	return "xc_user_roles"
}

func (RoleMenu) TableName() string {
	return "xc_role_menus"
}

func (OperationLog) TableName() string {
	return "xc_operation_logs"
}
