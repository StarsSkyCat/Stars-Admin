package file

import (
	"context"
	"time"

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
func (r repository) get(ctx context.Context, tx *gorm.DB, id model.ID) (file, error) {
	var row file
	err := r.conn(ctx, tx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
	return row, err
}
func (r repository) hash(ctx context.Context, owner model.ID, hash string) (file, error) {
	var row file
	err := r.conn(ctx, nil).First(&row, "created_by = ? AND storage = ? AND hash = ?", owner, "local", hash).Error
	return row, err
}
func (r repository) create(ctx context.Context, row *file, reservation model.ID) error {
	return r.conn(ctx, nil).Transaction(func(tx *gorm.DB) error {
		if err := r.conn(ctx, tx).Create(row).Error; err != nil {
			return err
		}
		return r.done(ctx, tx, reservation)
	})
}
func (r repository) remove(ctx context.Context, tx *gorm.DB, row file, task cleanup) error {
	if err := r.conn(ctx, tx).Create(&task).Error; err != nil {
		return err
	}
	return r.conn(ctx, tx).Delete(&row).Error
}
func (r repository) pending(ctx context.Context) ([]cleanup, error) {
	rows := []cleanup{}
	err := r.conn(ctx, nil).Where("next_attempt <= ?", time.Now().UTC()).Order("id").Limit(20).Find(&rows).Error
	return rows, err
}
func (r repository) lockCleanup(ctx context.Context, tx *gorm.DB, id model.ID) (cleanup, error) {
	var row cleanup
	err := r.conn(ctx, tx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
	return row, err
}
func (r repository) activePath(ctx context.Context, tx *gorm.DB, storage, path string) (bool, error) {
	var count int64
	err := r.conn(ctx, tx).Model(&file{}).Where("storage = ? AND path = ?", storage, path).Count(&count).Error
	return count > 0, err
}
func (r repository) done(ctx context.Context, tx *gorm.DB, id model.ID) error {
	return r.conn(ctx, tx).Delete(&cleanup{}, "id = ?", id).Error
}
func (r repository) retry(ctx context.Context, tx *gorm.DB, row cleanup) error {
	row.Attempts++
	row.NextAttempt = time.Now().UTC().Add(min(time.Hour, time.Duration(row.Attempts)*time.Minute))
	return r.conn(ctx, tx).Save(&row).Error
}
func (r repository) queue(ctx context.Context, row cleanup) error {
	return r.conn(ctx, nil).Create(&row).Error
}
