// Package table 导入受限 PostgreSQL 元数据并生成待审代码，不自动执行生成的 SQL。
package table

import (
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
)

type table struct {
	model.Base
	TableNameValue string `gorm:"column:table_name"`
	TableComment   string
	ModuleName     string
	FeatureName    string
	BusinessName   string
	Author         string
	GenType        string
	Options        string
	SyncedAt       *time.Time
}

func (table) TableName() string { return "gen_table" }

type column struct {
	model.Base
	TableID          model.ID
	ColumnName       string
	ColumnComment    string
	DBType           string
	GoType           string
	GoField          string
	JSONField        string
	IsPK             bool
	IsRequired       bool
	IsInsert         bool
	IsEdit           bool
	IsList           bool
	IsQuery          bool
	IsNullable       bool
	MaxLength        *int
	NumericPrecision *int
	NumericScale     *int
	ColumnDefault    *string
	QueryType        string
	HTMLType         string
	DictType         *string
	Sort             int
	ChangeKind       string
}

func (column) TableName() string { return "gen_table_column" }
