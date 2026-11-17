package config

import "github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"

// Input 不允许客户端修改内置标识。
type Input struct {
	Name   string  `json:"name" binding:"required,max=64"`
	Key    string  `json:"key" binding:"required,max=128"`
	Value  string  `json:"value" binding:"max=4096"`
	Remark *string `json:"remark" binding:"omitempty,max=500"`
}

// View 是参数管理公开资料，不用于保存秘密。
type View struct {
	httpx.Meta
	Input
	IsSystem bool `json:"isSystem"`
}

func view(r parameter) View {
	return View{ID: r.ID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Input: Input{r.Name, r.Key, r.Value, r.Remark}, IsSystem: r.IsSystem}
}
