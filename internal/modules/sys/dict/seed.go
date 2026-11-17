// Package dict 提供字典类型、数据、缓存与幂等初始化。
package dict

import (
	"context"
	"strconv"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type idSource interface{ Next() (int64, error) }
type dictionaryType struct {
	model.Base
	Name   string
	Type   string
	Status int
}

func (dictionaryType) TableName() string { return "sys_dict_type" }

type dictionaryData struct {
	model.Base
	DictType  string
	Label     string
	Value     string
	Sort      int
	Tag       *string
	IsDefault bool
	Status    int
}

func (dictionaryData) TableName() string { return "sys_dict_data" }

// Seed 只插入缺失键，不覆盖已有字典文本或状态。
func Seed(ctx context.Context, tx *gorm.DB, ids idSource) error {
	definitions := []struct {
		key, name string
		labels    []string
	}{{"sys_user_gender", "用户性别", []string{"未知", "男", "女"}}, {"sys_status", "状态", []string{"正常", "停用"}}, {"sys_menu_type", "菜单类型", []string{"目录", "菜单", "按钮"}}, {"sys_data_scope", "数据范围", []string{"全部", "自定义", "本部门", "本部门及以下", "仅本人"}}}
	for _, d := range definitions {
		id, err := ids.Next()
		if err != nil {
			return err
		}
		row := dictionaryType{ID: model.ID(id), Name: d.name, Type: d.key, Status: 1}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "type"}}, DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
		for i, label := range d.labels {
			n, err := ids.Next()
			if err != nil {
				return err
			}
			value := i + 1
			if d.key == "sys_user_gender" {
				value = i
			}
			entry := dictionaryData{ID: model.ID(n), DictType: d.key, Label: label, Value: strconv.Itoa(value), Sort: i, Status: 1}
			if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "dict_type"}, {Name: "value"}}, DoNothing: true}).Create(&entry).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
