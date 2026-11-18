package permission

import "github.com/StarsSkyCat/Stars-Admin/internal/core/model"

// Menu 是本人导航使用的菜单节点，只包含页面装配需要的字段。
type Menu struct {
	ID        model.ID `json:"id"`
	Name      string   `json:"name"`
	Type      int      `json:"type"`
	Path      *string  `json:"path"`
	Component *string  `json:"component"`
	Icon      *string  `json:"icon"`
	Sort      int      `json:"sort"`
	Visible   bool     `json:"visible"`
	Children  []Menu   `json:"children"`
}
