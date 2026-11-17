package user

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"gorm.io/gorm"
)

func (r repository) nameExists(ctx context.Context, tx *gorm.DB, name string) (bool, error) {
	var count int64
	err := r.conn(ctx, tx).Unscoped().Model(&user{}).Where("username = ?", name).Count(&count).Error
	return count > 0, err
}
func (r repository) export(ctx context.Context, q Query) ([]user, error) {
	db := userScope(ctx, r.conn(ctx, nil).Model(&user{}))
	if q.Username != "" {
		db = db.Where("username LIKE ?", "%"+q.Username+"%")
	}
	if q.Nickname != "" {
		db = db.Where("nickname LIKE ?", "%"+q.Nickname+"%")
	}
	if q.Status != 0 {
		db = db.Where("status = ?", q.Status)
	}
	if q.DeptID != nil {
		db = db.Where("dept_id = ?", *q.DeptID)
	}
	rows := []user{}
	if err := db.Order("id").Limit(10001).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) > 10000 {
		return nil, response.ErrInvalidParam
	}
	return rows, nil
}
