package menu

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct{ db *gorm.DB }

func (r repository) list(ctx context.Context, q Query, column string, desc bool) ([]menu, int64, error) {
	db := r.conn(ctx, nil).Model(&menu{})
	if q.Name != "" {
		db = db.Where("name LIKE ?", "%"+q.Name+"%")
	}
	if q.Status != 0 {
		db = db.Where("status = ?", q.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if column == "" {
		column = "sort"
	}
	var rows []menu
	err := db.Order(clause.OrderByColumn{Column: clause.Column{Name: column}, Desc: desc}).Order("id").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error
	return rows, total, err
}

func (r repository) conn(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx)
}
func (r repository) all(ctx context.Context, tx *gorm.DB) ([]menu, error) {
	var rows []menu
	err := r.conn(ctx, tx).Order("sort,id").Find(&rows).Error
	return rows, err
}
func (r repository) get(ctx context.Context, tx *gorm.DB, id model.ID) (menu, error) {
	var row menu
	err := r.conn(ctx, tx).First(&row, "id = ?", id).Error
	return row, err
}
func (r repository) save(ctx context.Context, tx *gorm.DB, row *menu, create bool) error {
	if create {
		return r.conn(ctx, tx).Create(row).Error
	}
	return r.conn(ctx, tx).Save(row).Error
}
func (r repository) remove(ctx context.Context, tx *gorm.DB, ids []model.ID, actor model.ID) error {
	if err := r.conn(ctx, tx).Model(&menu{}).Where("id IN ?", ids).Update("updated_by", actor).Error; err != nil {
		return err
	}
	return r.conn(ctx, tx).Delete(&menu{}, "id IN ?", ids).Error
}
