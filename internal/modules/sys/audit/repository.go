package audit

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct{ db *gorm.DB }

func (r repository) login(ctx context.Context, row *loginLog) error {
	return r.db.WithContext(ctx).Create(row).Error
}
func (r repository) operations(ctx context.Context, rows []operation) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}
func (r repository) list(ctx context.Context, q Query, oper bool) ([]operation, []loginLog, int64, error) {
	db := r.db.WithContext(ctx)
	if oper {
		db = db.Model(&operation{})
		if q.Username != "" {
			db = db.Where("username = ?", q.Username)
		}
	} else {
		db = db.Model(&loginLog{})
		if q.Username != "" {
			db = db.Where("username = ?", q.Username)
		}
	}
	if q.Status != 0 {
		db = db.Where("status = ?", q.Status)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return nil, nil, 0, err
	}
	if count > 10000 && q.export {
		return nil, nil, 0, response.ErrInvalidParam
	}
	if !q.export {
		db = db.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize)
	} else {
		db = db.Limit(10001)
	}
	operations := []operation{}
	logins := []loginLog{}
	db = db.Order("id DESC")
	var err error
	if oper {
		err = db.Find(&operations).Error
	} else {
		err = db.Find(&logins).Error
	}
	return operations, logins, count, err
}
func (r repository) clean(ctx context.Context, oper bool) error {
	if oper {
		return r.db.WithContext(ctx).Where("1 = 1").Delete(&operation{}).Error
	}
	return r.db.WithContext(ctx).Where("1 = 1").Delete(&loginLog{}).Error
}
