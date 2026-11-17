package user

import (
	"context"
	"errors"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/pkg/hash"
	"gorm.io/gorm"
)

func (r repository) seedAdmin(ctx context.Context, tx *gorm.DB) (bool, error) {
	var rows []user
	if err := r.conn(ctx, tx).Unscoped().Where("id = ? OR username = ?", 1, "admin").Find(&rows).Error; err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	if len(rows) != 1 || rows[0].ID != 1 || rows[0].Username != "admin" {
		return false, response.ErrConflict
	}
	return true, nil
}

// Seed 只在用户首次建立时读入初始密码并创建 admin 关系。
func (s *Service) Seed(ctx context.Context, tx *gorm.DB, deptID model.ID, password string) error {
	exists, err := s.repo.seedAdmin(ctx, tx)
	if err != nil || exists {
		return err
	}
	if password == "" {
		return errors.New("首次 seed 必须提供 STARS_SEED_ADMIN_PASSWORD")
	}
	encoded, err := hash.Password(password, s.cost)
	if err != nil {
		return err
	}
	row := user{ID: 1, Username: "admin", Nickname: "超级管理员", Password: encoded, DeptID: new(deptID), Status: 1, AuthVersion: 1}
	if err := s.repo.save(ctx, tx, &row, true); err != nil {
		return err
	}
	id, err := s.ids.Next()
	if err != nil {
		return err
	}
	return s.repo.replaceRoles(ctx, tx, 1, []userRole{{ID: model.ID(id), UserID: 1, RoleID: 1}})
}
