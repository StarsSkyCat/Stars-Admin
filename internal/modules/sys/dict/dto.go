package dict

import "github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"

// TypeInput 是类型编辑参数，标识创建后不可改名。
type TypeInput struct {
	Name   string  `json:"name" binding:"required,max=64"`
	Type   string  `json:"type" binding:"required,max=64"`
	Status int     `json:"status" binding:"required,oneof=1 2"`
	Remark *string `json:"remark" binding:"omitempty,max=500"`
}

// DataInput 是字典数据编辑参数。
type DataInput struct {
	DictType  string  `json:"dictType" binding:"required,max=64"`
	Label     string  `json:"label" binding:"required,max=64"`
	Value     string  `json:"value" binding:"required,max=64"`
	Sort      int     `json:"sort"`
	Tag       *string `json:"tag" binding:"omitempty,max=64"`
	IsDefault bool    `json:"isDefault"`
	Status    int     `json:"status" binding:"required,oneof=1 2"`
	Remark    *string `json:"remark" binding:"omitempty,max=500"`
}

// TypeView 是字典类型公开资料。
type TypeView struct {
	httpx.Meta
	TypeInput
}

// DataView 是字典项公开资料。
type DataView struct {
	httpx.Meta
	DataInput
}

func typeView(r dictionaryType) TypeView {
	return TypeView{ID: r.ID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, TypeInput: TypeInput{r.Name, r.Type, r.Status, r.Remark}}
}
func dataView(r dictionaryData) DataView {
	return DataView{ID: r.ID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DataInput: DataInput{r.DictType, r.Label, r.Value, r.Sort, r.Tag, r.IsDefault, r.Status, r.Remark}}
}
