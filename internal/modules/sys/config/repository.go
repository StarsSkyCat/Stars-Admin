package config

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct{ db *gorm.DB }

func (r repository) conn(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx)
}
func (r repository) keys(ctx context.Context, keys []string) ([]parameter, error) {
	var rows []parameter
	err := r.conn(ctx, nil).Where("key IN ?", keys).Find(&rows).Error
	return rows, err
}
func (r repository) get(ctx context.Context, tx *gorm.DB, id model.ID) (parameter, error) {
	var row parameter
	err := r.conn(ctx, tx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
	return row, err
}
func (r repository) byKey(ctx context.Context, key string) (parameter, error) {
	var row parameter
	err := r.conn(ctx, nil).First(&row, "key = ?", key).Error
	return row, err
}
func (r repository) list(ctx context.Context, q response.PageQuery) ([]parameter, int64, error) {
	db := r.conn(ctx, nil).Model(&parameter{})
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	rows := []parameter{}
	err := db.Order("id").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error
	return rows, count, err
}
func (r repository) save(ctx context.Context, tx *gorm.DB, row *parameter, create bool) error {
	if create {
		return r.conn(ctx, tx).Create(row).Error
	}
	return r.conn(ctx, tx).Save(row).Error
}
func (r repository) remove(ctx context.Context, tx *gorm.DB, ids []model.ID) error {
	return r.conn(ctx, tx).Delete(&parameter{}, "id IN ?", ids).Error
}
