package dept

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"gorm.io/gorm"
)

func (r repository) seedRoot(ctx context.Context, tx *gorm.DB) (department, bool, error) {
	var rows []department
	err := r.conn(ctx, tx).Unscoped().Where("parent_id = ? AND name = ?", 0, "总部").Order("id").Find(&rows).Error
	if err != nil {
		return department{}, false, err
	}
	if len(rows) > 0 {
		return rows[0], true, nil
	}
	return department{}, false, nil
}

// Seed 在外部事务中补齐根部门，不恢复已删除或已停用的节点。
func (s *Service) Seed(ctx context.Context, tx *gorm.DB) (model.ID, error) {
	row, exists, err := s.repo.seedRoot(ctx, tx)
	if err != nil {
		return 0, err
	}
	if exists {
		if row.DeletedAt.Valid {
			return 0, response.ErrConflict
		}
		return row.ID, nil
	}
	id, err := s.ids.Next()
	if err != nil {
		return 0, err
	}
	row = department{ID: model.ID(id), Name: "总部", Ancestors: "0", Status: 1}
	return row.ID, s.repo.save(ctx, tx, &row, true)
}
