package log

import (
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
)

// Query 只接受固定过滤字段。
type Query struct {
	response.PageQuery
	TaskID *model.ID `form:"taskId"`
	Status int       `form:"status"`
}

// View 的 Params 仅包含注册目标提供的脱敏摘要。
type View struct {
	ID        model.ID  `json:"id"`
	TaskID    model.ID  `json:"taskId"`
	Name      string    `json:"name"`
	TaskGroup string    `json:"taskGroup"`
	Target    string    `json:"target"`
	Params    string    `json:"params"`
	Status    int       `json:"status"`
	ErrorMsg  string    `json:"errorMsg"`
	CostMS    int       `json:"costMs"`
	StartedAt time.Time `json:"startedAt"`
}
