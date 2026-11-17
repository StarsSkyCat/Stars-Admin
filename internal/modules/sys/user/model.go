package user

import (
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
)

type user struct {
	model.Base
	Username     string
	Nickname     string
	Password     string
	Email        *string
	Phone        *string
	Gender       int
	Avatar       *string
	DeptID       *model.ID
	Status       int
	LoginIP      *string
	LoginAt      *time.Time
	PwdUpdatedAt *time.Time
	AuthVersion  int64
}

func (user) TableName() string { return "sys_user" }

type userRole struct {
	model.RelationBase
	UserID model.ID
	RoleID model.ID
}

func (userRole) TableName() string { return "sys_user_role" }
