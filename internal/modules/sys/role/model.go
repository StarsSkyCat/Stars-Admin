package role

import "github.com/StarsSkyCat/Stars-Admin/internal/core/model"

type role struct {
	model.Base
	Name      string
	Code      string
	Sort      int
	DataScope int
	Status    int
}

func (role) TableName() string { return "sys_role" }

type roleMenu struct {
	model.RelationBase
	RoleID model.ID
	MenuID model.ID
}

func (roleMenu) TableName() string { return "sys_role_menu" }

type roleDept struct {
	model.RelationBase
	RoleID model.ID
	DeptID model.ID
}

func (roleDept) TableName() string { return "sys_role_dept" }
