package post

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"gorm.io/gorm/clause"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"gorm.io/gorm"
)

type repository struct{ db *gorm.DB }

func (r repository) conn(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx)
}
func (r repository) get(ctx context.Context, tx *gorm.DB, id model.ID) (post, error) {
	var row post
	err := r.conn(ctx, tx).First(&row, "id = ?", id).Error
	return row, err
}

func (r repository) save(ctx context.Context, tx *gorm.DB, row *post, create bool) error {
	if create {
		return r.conn(ctx, tx).Create(row).Error
	}
	return r.conn(ctx, tx).Save(row).Error
}
func (r repository) remove(ctx context.Context, tx *gorm.DB, ids []model.ID, p model.ID) error {
	db := r.conn(ctx, tx)
	if err := db.Model(&post{}).Where("id IN ?", ids).Update("updated_by", p).Error; err != nil {
		return err
	}
	return db.Delete(&post{}, "id IN ?", ids).Error
}

func (r repository) list(ctx context.Context, q response.PageQuery, options bool, column string, desc bool) ([]post, int64, error) {
	db := r.conn(ctx, nil).Model(&post{})
	if options {
		db = db.Where("status = ?", 1)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if column == "" {
		column = "sort"
	}
	db = db.Order(clause.OrderByColumn{Column: clause.Column{Name: column}, Desc: desc}).Order("id")
	if !options {
		db = db.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize)
	}
	rows := []post{}
	err := db.Find(&rows).Error
	return rows, count, err
}
