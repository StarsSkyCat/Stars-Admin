package table

import (
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
)

// DBTable 是可导入的普通业务表摘要。
type DBTable struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
}

// ImportInput 只接受白名单内的表名，不能传 SQL。
type ImportInput struct {
	Tables []string `json:"tables" binding:"required,min=1,max=100"`
}

// ScopeConfig 必须显式选择 admin 或映射列；默认 deny。
type ScopeConfig struct {
	Mode             string `json:"mode"`
	DepartmentColumn string `json:"departmentColumn"`
	OwnerColumn      string `json:"ownerColumn"`
}

// ColumnInput 只编辑生成行为与合法名称，不能覆盖数据库类型快照。
type ColumnInput struct {
	ColumnName string  `json:"columnName" binding:"required,max=128"`
	GoField    string  `json:"goField" binding:"required,max=64"`
	JSONField  string  `json:"jsonField" binding:"required,max=64"`
	IsRequired bool    `json:"isRequired"`
	IsInsert   bool    `json:"isInsert"`
	IsEdit     bool    `json:"isEdit"`
	IsList     bool    `json:"isList"`
	IsQuery    bool    `json:"isQuery"`
	QueryType  string  `json:"queryType"`
	HTMLType   string  `json:"htmlType"`
	DictType   *string `json:"dictType" binding:"omitempty,max=64"`
}

// Input 不接收原始 options/snapshot 或任意模板。
type Input struct {
	ModuleName   string        `json:"moduleName" binding:"required,max=64"`
	FeatureName  string        `json:"featureName" binding:"required,max=64"`
	BusinessName string        `json:"businessName" binding:"required,max=64"`
	Author       string        `json:"author" binding:"max=64"`
	GenType      string        `json:"genType" binding:"required,oneof=zip local"`
	Scope        ScopeConfig   `json:"scope"`
	Columns      []ColumnInput `json:"columns" binding:"required,max=128,dive"`
}

// ColumnView 分离公开配置与持久化字段。
type ColumnView struct {
	ID model.ID `json:"id"`
	ColumnInput
	DBType           string  `json:"dbType"`
	GoType           string  `json:"goType"`
	IsPK             bool    `json:"isPk"`
	IsNullable       bool    `json:"isNullable"`
	MaxLength        *int    `json:"maxLength"`
	NumericPrecision *int    `json:"numericPrecision"`
	NumericScale     *int    `json:"numericScale"`
	ColumnDefault    *string `json:"columnDefault"`
	ChangeKind       string  `json:"changeKind"`
	Removed          bool    `json:"removed"`
}

// View 是生成配置，结构快照通过单独元数据展示。
type View struct {
	httpx.Meta
	TableName    string       `json:"tableName"`
	TableComment string       `json:"tableComment"`
	ModuleName   string       `json:"moduleName"`
	FeatureName  string       `json:"featureName"`
	BusinessName string       `json:"businessName"`
	Author       string       `json:"author"`
	GenType      string       `json:"genType"`
	Scope        ScopeConfig  `json:"scope"`
	Columns      []ColumnView `json:"columns"`
	SyncedAt     *time.Time   `json:"syncedAt"`
}

// GeneratedFile 仅包含安全相对路径和待审文本。
type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// GenerateResult 是 local 输出结果，不返回服务器绝对目录。
type GenerateResult struct {
	Files []string `json:"files"`
}

// Generated 包含调用方可选择的 zip 或本地结果。
type Generated struct {
	ZIP   []byte
	Local *GenerateResult
}

// MetadataColumn 保留 PG 类型、可空性、默认值与主键，不能由 HTTP 覆盖。
type MetadataColumn struct {
	Name      string  `json:"name"`
	Comment   string  `json:"comment"`
	DBType    string  `json:"dbType"`
	UDT       string  `json:"udt"`
	Nullable  bool    `json:"nullable"`
	MaxLength *int    `json:"maxLength"`
	Precision *int    `json:"precision"`
	Scale     *int    `json:"scale"`
	Default   *string `json:"default"`
	Primary   bool    `json:"primary"`
	Identity  bool    `json:"identity"`
	Generated bool    `json:"generated"`
	Position  int     `json:"position"`
}

// Snapshot 是仅从数据库采集的结构事实。
type Snapshot struct {
	Name        string           `json:"name"`
	Comment     string           `json:"comment"`
	Columns     []MetadataColumn `json:"columns"`
	Constraints []string         `json:"constraints"`
	Indexes     []string         `json:"indexes"`
}
type options struct {
	Snapshot   Snapshot    `json:"snapshot"`
	Scope      ScopeConfig `json:"scope"`
	Configured bool        `json:"configured"`
}
