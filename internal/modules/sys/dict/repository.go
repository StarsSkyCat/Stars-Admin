package dict

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"gorm.io/gorm"
)

type repository struct{ db *gorm.DB }

func (r repository) conn(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx)
}
func (r repository) lock(ctx context.Context, tx *gorm.DB) error { // 固定表名的事务锁串行化类型删除与数据写入，属于字典引用完整性例外。
	return r.conn(ctx, tx).Exec("LOCK TABLE sys_dict_type IN SHARE ROW EXCLUSIVE MODE").Error
}
func (r repository) typeGet(ctx context.Context, tx *gorm.DB, id model.ID) (dictionaryType, error) {
	var row dictionaryType
	err := r.conn(ctx, tx).First(&row, "id = ?", id).Error
	return row, err
}
func (r repository) typeKey(ctx context.Context, tx *gorm.DB, key string) (dictionaryType, error) {
	var row dictionaryType
	err := r.conn(ctx, tx).First(&row, "type = ?", key).Error
	return row, err
}
func (r repository) dataGet(ctx context.Context, tx *gorm.DB, id model.ID) (dictionaryData, error) {
	var row dictionaryData
	err := r.conn(ctx, tx).First(&row, "id = ?", id).Error
	return row, err
}
func (r repository) list(ctx context.Context, q response.PageQuery, data bool, key string) ([]dictionaryType, []dictionaryData, int64, error) {
	db := r.conn(ctx, nil)
	var count int64
	if data {
		db = db.Model(&dictionaryData{})
		if key != "" {
			db = db.Where("dict_type = ?", key)
		}
	} else {
		db = db.Model(&dictionaryType{})
	}
	if err := db.Count(&count).Error; err != nil {
		return nil, nil, 0, err
	}
	db = db.Order("id").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize)
	types := []dictionaryType{}
	entries := []dictionaryData{}
	var err error
	if data {
		err = db.Find(&entries).Error
	} else {
		err = db.Find(&types).Error
	}
	return types, entries, count, err
}
func (r repository) enabled(ctx context.Context, key string) ([]dictionaryData, error) {
	rows := []dictionaryData{}
	err := r.conn(ctx, nil).Where("dict_type = ? AND status = ?", key, 1).Order("sort,id").Find(&rows).Error
	return rows, err
}
func (r repository) save(ctx context.Context, tx *gorm.DB, row any, create bool) error {
	if create {
		return r.conn(ctx, tx).Create(row).Error
	}
	return r.conn(ctx, tx).Save(row).Error
}
func (r repository) removeType(ctx context.Context, tx *gorm.DB, ids []model.ID, keys []string, p model.ID) error {
	db := r.conn(ctx, tx)
	if err := db.Model(&dictionaryData{}).Where("dict_type IN ?", keys).Update("updated_by", p).Error; err != nil {
		return err
	}
	if err := db.Delete(&dictionaryData{}, "dict_type IN ?", keys).Error; err != nil {
		return err
	}
	if err := db.Model(&dictionaryType{}).Where("id IN ?", ids).Update("updated_by", p).Error; err != nil {
		return err
	}
	return db.Delete(&dictionaryType{}, "id IN ?", ids).Error
}
func (r repository) removeData(ctx context.Context, tx *gorm.DB, ids []model.ID, p model.ID) error {
	db := r.conn(ctx, tx)
	if err := db.Model(&dictionaryData{}).Where("id IN ?", ids).Update("updated_by", p).Error; err != nil {
		return err
	}
	return db.Delete(&dictionaryData{}, "id IN ?", ids).Error
}
