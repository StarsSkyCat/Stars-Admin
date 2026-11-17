package menu

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"gorm.io/gorm"
)

func (r repository) seedFind(ctx context.Context, tx *gorm.DB, parent model.ID, in Input) (menu, bool, error) {
	var rows []menu
	db := r.conn(ctx, tx).Unscoped()
	if in.Perm != nil {
		db = db.Where("perm = ?", *in.Perm)
	} else {
		db = db.Where("parent_id = ? AND path = ?", parent, *in.Path)
	}
	err := db.Order("id").Find(&rows).Error
	if len(rows) > 0 {
		return rows[0], true, err
	}
	return menu{}, false, err
}

// Seed 仅补齐 M1—M4 已实现功能的菜单和按钮，不改写已有节点或授权。
func (s *Service) Seed(ctx context.Context, tx *gorm.DB) ([]model.ID, error) {
	seedNode := func(parent model.ID, in Input) (menu, error) {
		row, exists, err := s.repo.seedFind(ctx, tx, parent, in)
		if err != nil || exists {
			return row, err
		}
		id, err := s.ids.Next()
		if err != nil {
			return row, err
		}
		row = menu{ID: model.ID(id), ParentID: parent, Name: in.Name, Type: in.Type, Path: in.Path, Perm: in.Perm, Visible: 1, Status: 1}
		return row, s.repo.save(ctx, tx, &row, true)
	}
	root, err := seedNode(0, Input{Name: "系统管理", Type: 1, Path: new("/system")})
	if err != nil {
		return nil, err
	}
	ids := []model.ID{}
	if root.DeletedAt.Valid {
		return ids, nil
	}
	ids = append(ids, root.ID)
	features := []struct{ code, name string }{{"user", "用户管理"}, {"role", "角色管理"}, {"menu", "菜单管理"}, {"dept", "部门管理"}, {"post", "岗位管理"}, {"dict", "字典管理"}, {"config", "参数管理"}, {"file", "文件管理"}, {"audit", "日志审计"}}
	for _, feature := range features {
		node, err := seedNode(root.ID, Input{Name: feature.name, Type: 2, Path: new("/system/" + feature.code)})
		if err != nil {
			return nil, err
		}
		if node.DeletedAt.Valid {
			continue
		}
		ids = append(ids, node.ID)
		actions := []string{"list", "get", "add", "edit", "remove"}
		if feature.code == "user" {
			actions = append(actions, "resetPwd", "grant", "import", "export")
		}
		if feature.code == "config" || feature.code == "dict" {
			actions = append(actions, "refresh")
		}
		if feature.code == "file" {
			actions = []string{"add", "get", "remove"}
		}
		if feature.code == "audit" {
			actions = []string{"operList", "operClean", "operExport", "loginList", "loginClean", "loginExport", "onlineList", "forceLogout"}
		}
		if feature.code == "role" {
			actions = append(actions, "grant")
		}
		for _, action := range actions {
			button, err := seedNode(node.ID, Input{Name: feature.name + "-" + action, Type: 3, Perm: new("sys:" + feature.code + ":" + action)})
			if err != nil {
				return nil, err
			}
			if !button.DeletedAt.Valid {
				ids = append(ids, button.ID)
			}
		}
	}

	// 仅补齐已实现的工具权限节点，不重置任何现有角色授权。
	tools := []struct {
		module, feature, name string
		actions               []string
	}{{"job", "task", "定时任务", []string{"list", "get", "add", "edit", "remove", "run"}}, {"job", "log", "任务日志", []string{"list", "remove"}}, {"gen", "table", "代码生成", []string{"list", "get", "import", "edit", "remove", "preview", "gen"}}}
	for _, tool := range tools {
		node, err := seedNode(0, Input{Name: tool.name, Type: 2, Path: new("/" + tool.module + "/" + tool.feature)})
		if err != nil {
			return nil, err
		}
		if node.DeletedAt.Valid {
			continue
		}
		ids = append(ids, node.ID)
		for _, action := range tool.actions {
			button, err := seedNode(node.ID, Input{Name: tool.name + "-" + action, Type: 3, Perm: new(tool.module + ":" + tool.feature + ":" + action)})
			if err != nil {
				return nil, err
			}
			if !button.DeletedAt.Valid {
				ids = append(ids, button.ID)
			}
		}
	}
	return ids, nil
}
