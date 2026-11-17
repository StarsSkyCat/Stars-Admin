package task

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
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
func (r repository) get(ctx context.Context, tx *gorm.DB, id model.ID) (task, error) {
	var row task
	err := r.conn(ctx, tx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
	return row, err
}
func (r repository) enabled(ctx context.Context) ([]task, error) {
	rows := []task{}
	err := r.conn(ctx, nil).Where("status = ?", 1).Find(&rows).Error
	return rows, err
}
func (r repository) save(ctx context.Context, tx *gorm.DB, row *task, create bool) error {
	if create {
		return r.conn(ctx, tx).Create(row).Error
	}
	return r.conn(ctx, tx).Save(row).Error
}
func (r repository) list(ctx context.Context, q Query, column string, desc bool) ([]task, int64, error) {
	db := r.conn(ctx, nil).Model(&task{})
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
	rows := []task{}
	err := db.Order(clause.OrderByColumn{Column: clause.Column{Name: column}, Desc: desc}).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error
	return rows, total, err
}
func (r repository) remove(ctx context.Context, tx *gorm.DB, ids []model.ID, p model.ID) error {
	db := r.conn(ctx, tx)
	if err := db.Model(&task{}).Where("id IN ?", ids).Update("updated_by", p).Error; err != nil {
		return err
	}
	return db.Delete(&task{}, "id IN ?", ids).Error
}
