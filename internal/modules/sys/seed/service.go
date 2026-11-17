// Package seed 编排跨功能种子事务，不读取其他功能模型或数据表。
package seed

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/permission"
	"gorm.io/gorm"
)

// Departments 提供根部门的幂等初始化能力。
type Departments interface {
	Seed(context.Context, *gorm.DB) (model.ID, error)
}

// Menus 提供菜单种子能力。
type Menus interface {
	Seed(context.Context, *gorm.DB) ([]model.ID, error)
}

// Roles 提供保留角色初始化能力。
type Roles interface {
	Seed(context.Context, *gorm.DB, []model.ID) error
}

// Users 提供保留用户初始化能力。
type Users interface {
	Seed(context.Context, *gorm.DB, model.ID, string) error
}

// Run 在单个事务执行所有初始化及策略投影，失败整体回滚。
func Run(ctx context.Context, db *gorm.DB, depts Departments, menus Menus, roles Roles, users Users, password string, extras func(context.Context, *gorm.DB) error, project func(context.Context, *gorm.DB) ([]permission.Rule, error), policies *permission.Store) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := policies.Lock(ctx, tx); err != nil {
			return err
		}
		deptID, err := depts.Seed(ctx, tx)
		if err != nil {
			return err
		}
		menuIDs, err := menus.Seed(ctx, tx)
		if err != nil {
			return err
		}
		if err := roles.Seed(ctx, tx, menuIDs); err != nil {
			return err
		}
		if err := users.Seed(ctx, tx, deptID, password); err != nil {
			return err
		}
		if err := extras(ctx, tx); err != nil {
			return err
		}
		rules, err := project(ctx, tx)
		if err != nil {
			return err
		}
		return policies.Replace(ctx, tx, rules)
	})
}
