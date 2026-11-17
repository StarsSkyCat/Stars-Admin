package dept

import "github.com/StarsSkyCat/Stars-Admin/internal/core/model"

type department struct {
	model.Base
	ParentID  model.ID
	Ancestors string
	Name      string
	Sort      int
	Leader    *string
	Phone     *string
	Status    int
}

func (department) TableName() string { return "sys_dept" }
