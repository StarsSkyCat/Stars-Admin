package config

import "github.com/StarsSkyCat/Stars-Admin/internal/core/model"

type parameter struct {
	model.Base
	Name     string
	Key      string
	Value    string
	IsSystem bool
}

func (parameter) TableName() string { return "sys_config" }
