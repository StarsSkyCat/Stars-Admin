package dict

import "github.com/StarsSkyCat/Stars-Admin/internal/core/model"

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
