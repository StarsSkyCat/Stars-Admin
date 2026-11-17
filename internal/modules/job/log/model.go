package log

import (
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
)

type entry struct {
	model.LogBase
	TaskID    model.ID
	Name      string
	TaskGroup string
	Target    string
	Params    string
	Status    int
	ErrorMsg  string
	CostMS    int
	StartedAt time.Time
}

func (entry) TableName() string { return "job_log" }
