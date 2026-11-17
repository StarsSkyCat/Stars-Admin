package menu

import "github.com/StarsSkyCat/Stars-Admin/internal/core/model"

type menu struct {
	model.Base
	ParentID  model.ID
	Name      string
	Type      int
	Path      *string
	Component *string
	Perm      *string
	Icon      *string
	Sort      int
	Visible   int
	Status    int
}

func (menu) TableName() string { return "sys_menu" }
