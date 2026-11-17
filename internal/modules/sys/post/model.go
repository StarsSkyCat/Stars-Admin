// Package post 提供岗位管理与有效岗位引用校验。
package post

import "github.com/StarsSkyCat/Stars-Admin/internal/core/model"

type post struct {
	model.Base
	Code   string
	Name   string
	Sort   int
	Status int
}

func (post) TableName() string { return "sys_post" }
