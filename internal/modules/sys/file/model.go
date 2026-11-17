package file

import (
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
)

type file struct {
	model.Base
	Name         string
	OriginalName string
	Path         string
	Size         int64
	Mime         string
	Storage      string
	Hash         string
}

func (file) TableName() string { return "sys_file" }

type cleanup struct {
	model.RelationBase
	Path        string
	Storage     string
	Attempts    int
	NextAttempt time.Time
}

func (cleanup) TableName() string { return "sys_file_cleanup" }
