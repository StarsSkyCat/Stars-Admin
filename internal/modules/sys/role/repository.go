package role

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
func (r repository) get(ctx context.Context, tx *gorm.DB, id model.ID) (role, error) {
	var row role
	err := r.conn(ctx, tx).First(&row, "id = ?", id).Error
	return row, err
}
func (r repository) list(ctx context.Context, q Query) ([]role, int64, error) {
	db := r.conn(ctx, nil).Model(&role{})
	if q.Name != "" {
		db = db.Where("name LIKE ?", "%"+q.Name+"%")
	}
	if q.Code != "" {
		db = db.Where("code = ?", q.Code)
	}
	if q.Status != 0 {
		db = db.Where("status = ?", q.Status)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	column, desc, err := q.Sort(map[string]string{"id": "id", "sort": "sort", "name": "name", "createdAt": "created_at"})
	if err != nil {
		return nil, 0, err
	}
	if column == "" {
		column = "id"
	}
	var rows []role
	err = db.Order(clause.OrderByColumn{Column: clause.Column{Name: column}, Desc: desc}).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error
	return rows, count, err
}
func (r repository) save(ctx context.Context, tx *gorm.DB, row *role, create bool) error {
	if create {
		return r.conn(ctx, tx).Create(row).Error
	}
	return r.conn(ctx, tx).Save(row).Error
}
func (r repository) active(ctx context.Context, tx *gorm.DB) ([]role, error) {
	var rows []role
	err := r.conn(ctx, tx).Where("status = ?", 1).Find(&rows).Error
	return rows, err
}
func (r repository) menus(ctx context.Context, tx *gorm.DB, ids []model.ID) ([]roleMenu, error) {
	var rows []roleMenu
	db := r.conn(ctx, tx)
	if ids != nil {
		db = db.Where("role_id IN ?", ids)
	}
	err := db.Find(&rows).Error
	return rows, err
}
func (r repository) depts(ctx context.Context, tx *gorm.DB, id model.ID) ([]model.ID, error) {
	var ids []model.ID
	err := r.conn(ctx, tx).Model(&roleDept{}).Where("role_id = ?", id).Pluck("dept_id", &ids).Error
	return ids, err
}
func (r repository) replaceMenus(ctx context.Context, tx *gorm.DB, id model.ID, rows []roleMenu) error {
	if err := r.conn(ctx, tx).Where("role_id = ?", id).Delete(&roleMenu{}).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	return r.conn(ctx, tx).Create(&rows).Error
}
func (r repository) replaceDepts(ctx context.Context, tx *gorm.DB, id model.ID, rows []roleDept) error {
	if err := r.conn(ctx, tx).Where("role_id = ?", id).Delete(&roleDept{}).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	return r.conn(ctx, tx).Create(&rows).Error
}
func (r repository) removeMenus(ctx context.Context, tx *gorm.DB, ids []model.ID) error {
	return r.conn(ctx, tx).Where("menu_id IN ?", ids).Delete(&roleMenu{}).Error
}
func (r repository) remove(ctx context.Context, tx *gorm.DB, ids []model.ID, actor model.ID) error {
	db := r.conn(ctx, tx)
	if err := db.Where("role_id IN ?", ids).Delete(&roleMenu{}).Error; err != nil {
		return err
	}
	if err := db.Where("role_id IN ?", ids).Delete(&roleDept{}).Error; err != nil {
		return err
	}
	if err := db.Model(&role{}).Where("id IN ?", ids).Update("updated_by", actor).Error; err != nil {
		return err
	}
	return db.Delete(&role{}, "id IN ?", ids).Error
}
