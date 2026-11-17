package user

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"gorm.io/gorm"
)

func (r repository) posts(ctx context.Context, tx *gorm.DB, id model.ID) ([]model.ID, error) {
	ids := []model.ID{}
	err := r.conn(ctx, tx).Model(&userPost{}).Where("user_id = ?", id).Pluck("post_id", &ids).Error
	return ids, err
}
func (r repository) replacePosts(ctx context.Context, tx *gorm.DB, id model.ID, rows []userPost) error {
	if err := r.conn(ctx, tx).Where("user_id = ?", id).Delete(&userPost{}).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	return r.conn(ctx, tx).Create(&rows).Error
}
func (r repository) hasPosts(ctx context.Context, tx *gorm.DB, ids []model.ID) (bool, error) {
	var count int64
	err := r.conn(ctx, tx).Model(&userPost{}).Where("post_id IN ?", ids).Count(&count).Error
	return count > 0, err
}
