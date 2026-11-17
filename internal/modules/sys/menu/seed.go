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

// Seed 仅补齐 M2 已实现功能的菜单和按钮，不改写已有节点或授权。
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
	features := []struct{ code, name string }{{"user", "用户管理"}, {"role", "角色管理"}, {"menu", "菜单管理"}, {"dept", "部门管理"}}
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
			actions = append(actions, "resetPwd", "grant")
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
	return ids, nil
}
