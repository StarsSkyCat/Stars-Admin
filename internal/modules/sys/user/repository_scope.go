package user

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

func userScope(ctx context.Context, db *gorm.DB) *gorm.DB {
	if scope, ok := security.CurrentScope(ctx); ok {
		return scope.Apply(db, "dept_id", "id", false)
	}
	return db
}
func (r repository) department(ctx context.Context, tx *gorm.DB, id model.ID) (*model.ID, error) {
	var row user
	err := r.conn(ctx, tx).Select("id,dept_id").First(&row, "id = ?", id).Error
	return row.DeptID, err
}

func (r repository) identity(ctx context.Context, tx *gorm.DB, id model.ID) (user, error) {
	var row user
	err := r.conn(ctx, tx).First(&row, "id = ?", id).Error
	return row, err
}
