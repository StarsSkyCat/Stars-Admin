package user

import (
	"context"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
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
func (r repository) get(ctx context.Context, tx *gorm.DB, id model.ID) (user, error) {
	var row user
	err := r.conn(ctx, tx).First(&row, "id = ?", id).Error
	return row, err
}
func (r repository) byName(ctx context.Context, name string) (user, error) {
	var row user
	err := r.conn(ctx, nil).First(&row, "username = ?", name).Error
	return row, err
}
func (r repository) list(ctx context.Context, q Query) ([]user, int64, error) {
	db := r.conn(ctx, nil).Model(&user{})
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
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	column, desc, err := q.Sort(map[string]string{"id": "id", "username": "username", "createdAt": "created_at"})
	if err != nil {
		return nil, 0, err
	}
	if column == "" {
		column = "id"
	}
	var rows []user
	err = db.Order(clause.OrderByColumn{Column: clause.Column{Name: column}, Desc: desc}).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error
	return rows, count, err
}
func (r repository) save(ctx context.Context, tx *gorm.DB, row *user, create bool) error {
	if create {
		return r.conn(ctx, tx).Create(row).Error
	}
	return r.conn(ctx, tx).Save(row).Error
}
func (r repository) roles(ctx context.Context, tx *gorm.DB, id model.ID) ([]model.ID, error) {
	ids := []model.ID{}
	err := r.conn(ctx, tx).Model(&userRole{}).Where("user_id = ?", id).Pluck("role_id", &ids).Error
	return ids, err
}
func (r repository) replaceRoles(ctx context.Context, tx *gorm.DB, id model.ID, rows []userRole) error {
	if err := r.conn(ctx, tx).Where("user_id = ?", id).Delete(&userRole{}).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	return r.conn(ctx, tx).Create(&rows).Error
}
func (r repository) hasDepartments(ctx context.Context, tx *gorm.DB, ids []model.ID) (bool, error) {
	var count int64
	err := r.conn(ctx, tx).Model(&user{}).Where("dept_id IN ?", ids).Count(&count).Error
	return count > 0, err
}
func (r repository) hasRoles(ctx context.Context, tx *gorm.DB, ids []model.ID) (bool, error) {
	var count int64
	err := r.conn(ctx, tx).Model(&userRole{}).Where("role_id IN ?", ids).Count(&count).Error
	return count > 0, err
}
func (r repository) roleUsers(ctx context.Context, tx *gorm.DB, ids []model.ID) ([]model.ID, error) {
	var out []model.ID
	err := r.conn(ctx, tx).Model(&userRole{}).Where("role_id IN ?", ids).Distinct("user_id").Pluck("user_id", &out).Error
	return out, err
}
func (r repository) bump(ctx context.Context, tx *gorm.DB, ids []model.ID) ([]security.Revocation, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	db := r.conn(ctx, tx)
	if err := db.Model(&user{}).Where("id IN ?", ids).Update("auth_version", gorm.Expr("auth_version + 1")).Error; err != nil {
		return nil, err
	}
	var rows []user
	if err := db.Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := []security.Revocation{}
	for _, row := range rows {
		out = append(out, security.Revocation{UserID: int64(row.ID), Version: row.AuthVersion})
	}
	return out, nil
}
func (r repository) remove(ctx context.Context, tx *gorm.DB, ids []model.ID, actor model.ID) error {
	db := r.conn(ctx, tx)
	if err := db.Where("user_id IN ?", ids).Delete(&userRole{}).Error; err != nil {
		return err
	}
	if err := db.Model(&user{}).Where("id IN ?", ids).Update("updated_by", actor).Error; err != nil {
		return err
	}
	return db.Delete(&user{}, "id IN ?", ids).Error
}
func (r repository) members(ctx context.Context, tx *gorm.DB) ([]userRole, []model.ID, error) {
	db := r.conn(ctx, tx)
	var rows []userRole
	var ids []model.ID
	if err := db.Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	err := db.Model(&user{}).Where("status = ?", 1).Pluck("id", &ids).Error
	return rows, ids, err
}
func (r repository) login(ctx context.Context, id model.ID, ip string) error {
	return r.conn(ctx, nil).Model(&user{}).Where("id = ?", id).Updates(map[string]any{"login_ip": ip, "login_at": time.Now().UTC()}).Error
}
