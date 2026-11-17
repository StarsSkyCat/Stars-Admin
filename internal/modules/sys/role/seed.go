package role

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"gorm.io/gorm"
)

func (r repository) seedAdmin(ctx context.Context, tx *gorm.DB) (bool, error) {
	var rows []role
	if err := r.conn(ctx, tx).Unscoped().Where("id = ? OR code = ?", 1, "admin").Find(&rows).Error; err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	if len(rows) != 1 || rows[0].ID != 1 || rows[0].Code != "admin" {
		return false, response.ErrConflict
	}
	return true, nil
}

// Seed 只在保留角色首次建立时创建其菜单授权，重复执行不覆盖授权。
func (s *Service) Seed(ctx context.Context, tx *gorm.DB, menuIDs []model.ID) error {
	exists, err := s.repo.seedAdmin(ctx, tx)
	if err != nil || exists {
		return err
	}
	row := role{ID: 1, Name: "超级管理员", Code: "admin", DataScope: 1, Status: 1}
	if err := s.repo.save(ctx, tx, &row, true); err != nil {
		return err
	}
	rows := []roleMenu{}
	for _, menuID := range menuIDs {
		id, err := s.ids.Next()
		if err != nil {
			return err
		}
		rows = append(rows, roleMenu{ID: model.ID(id), RoleID: 1, MenuID: menuID})
	}
	return s.repo.replaceMenus(ctx, tx, 1, rows)
}
