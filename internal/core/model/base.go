// Package model 定义持久化公共字段，不直接用作 HTTP DTO。
package model

import (
	"encoding/json/v2"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// ID 在数据库中保留 int64，在 JSON 边界使用十进制字符串。
type ID int64

func (id ID) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, strconv.FormatInt(int64(id), 10)), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return errors.New("ID 必须为十进制字符串")
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil || value < 0 {
		return errors.New("ID 超出有效范围")
	}
	*id = ID(value)
	return nil
}

// RelationBase 是物理解绑的关联表公共字段。
type RelationBase struct {
	ID        ID        `gorm:"primaryKey;autoIncrement:false"`
	CreatedBy ID        `gorm:"not null;default:0"`
	UpdatedBy ID        `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// Base 是普通实体的软删除公共字段。
type Base struct {
	RelationBase
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Remark    *string        `gorm:"size:500"`
}

// LogBase 是追加日志的公共字段，不含更新或软删除字段。
type LogBase struct {
	ID        ID        `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt time.Time `gorm:"not null"`
}
