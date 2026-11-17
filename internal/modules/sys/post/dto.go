package post

import "github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"

// Input 是岗位业务参数。
type Input struct {
	Code   string  `json:"code" binding:"required,max=64"`
	Name   string  `json:"name" binding:"required,max=64"`
	Sort   int     `json:"sort"`
	Status int     `json:"status" binding:"required,oneof=1 2"`
	Remark *string `json:"remark" binding:"omitempty,max=500"`
}

// View 是岗位公开资料。
type View struct {
	httpx.Meta
	Input
}

func view(p post) View {
	return View{ID: p.ID, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, Input: Input{p.Code, p.Name, p.Sort, p.Status, p.Remark}}
}
