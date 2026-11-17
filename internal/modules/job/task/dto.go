package task

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/scheduler"
)

// Input 不允许从通用编辑入口绕过启停规则。
type Input struct {
	Name          string  `json:"name" binding:"required,max=64"`
	TaskGroup     string  `json:"taskGroup" binding:"max=64"`
	Cron          string  `json:"cron" binding:"required,max=64"`
	Target        string  `json:"target" binding:"required,max=255"`
	Params        string  `json:"params" binding:"max=4096"`
	MisfirePolicy int     `json:"misfirePolicy" binding:"oneof=0 1 2"`
	Concurrent    bool    `json:"concurrent"`
	Remark        *string `json:"remark" binding:"omitempty,max=500"`
}

// View 是任务公开资料，不包含持久化模型。
type View struct {
	httpx.Meta
	Input
	Status int `json:"status"`
}

// Query 是任务列表白名单过滤项。
type Query struct {
	response.PageQuery
	Name   string `form:"name"`
	Status int    `form:"status"`
}

// StatusInput 仅启停接口接收状态。
type StatusInput struct {
	Status int `json:"status" binding:"required,oneof=1 2"`
}

// RunResult 区分投递成功与门禁跳过，不伪报任务完成。
type RunResult struct {
	State string `json:"state"`
}

func view(r task) View {
	return View{ID: r.ID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Input: Input{r.Name, r.TaskGroup, r.Cron, r.Target, r.Params, r.MisfirePolicy, r.Concurrent, r.Remark}, Status: r.Status}
}
func spec(r task) scheduler.Spec {
	return scheduler.Spec{ID: int64(r.ID), Name: r.Name, Group: r.TaskGroup, Cron: r.Cron, Target: r.Target, Params: r.Params, Concurrent: r.Concurrent, Misfire: r.MisfirePolicy}
}
