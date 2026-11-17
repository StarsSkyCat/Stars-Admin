package dept

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
)

func buildTree(rows []department, parent model.ID) []View {
	out := []View{}
	for _, row := range rows {
		if row.ParentID == parent {
			v := view(row)
			v.Children = buildTree(rows, row.ID)
			out = append(out, v)
		}
	}
	return out
}
