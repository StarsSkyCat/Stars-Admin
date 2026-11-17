// Package task 提供任务配置、调度同步及手动执行能力。
package task

import "github.com/StarsSkyCat/Stars-Admin/internal/core/model"

type task struct {
	model.Base
	Name          string
	TaskGroup     string
	Cron          string
	Target        string
	Params        string
	MisfirePolicy int
	Concurrent    bool
	Status        int
}

func (task) TableName() string { return "job_task" }
