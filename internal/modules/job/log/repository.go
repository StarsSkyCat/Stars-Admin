package log

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct{ db *gorm.DB }

func (r repository) write(ctx context.Context, rows []entry) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}
func (r repository) list(ctx context.Context, q Query) ([]entry, int64, error) {
	db := r.db.WithContext(ctx).Model(&entry{})
	if q.TaskID != nil {
		db = db.Where("task_id = ?", *q.TaskID)
	}
	if q.Status != 0 {
		db = db.Where("status = ?", q.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []entry{}
	err := db.Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error
	return rows, total, err
}
func (r repository) clean(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("1 = 1").Delete(&entry{}).Error
}
