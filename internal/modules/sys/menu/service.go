package menu

import (
	"context"
	"regexp"
	"strings"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

type idSource interface{ Next() (int64, error) }
type changeGate interface {
	Change(context.Context, func(*gorm.DB) ([]security.Revocation, error)) error
}

type menuStore interface {
	list(context.Context, Query, string, bool) ([]menu, int64, error)
	all(context.Context, *gorm.DB) ([]menu, error)
	get(context.Context, *gorm.DB, model.ID) (menu, error)
	save(context.Context, *gorm.DB, *menu, bool) error
	remove(context.Context, *gorm.DB, []model.ID, model.ID) error
	seedFind(context.Context, *gorm.DB, model.ID, Input) (menu, bool, error)
}

// Roles 在菜单删除事务内物理解绑引用，避免功能直接访问其他表。
type Roles interface {
	RemoveMenus(context.Context, *gorm.DB, []model.ID) error
}

// Service 管理树结构并提供可重建策略的有效权限快照。
type Service struct {
	repo  menuStore
	ids   idSource
	gate  changeGate
	roles Roles
}

// New 创建菜单业务。
func New(db *gorm.DB, ids idSource, gate changeGate) *Service {
	return &Service{repo: repository{db}, ids: ids, gate: gate}
}

// SetRoles 完成功能间引用装配。
func (s *Service) SetRoles(roles Roles) { s.roles = roles }

// List 使用明确的排序字段映射，不拼接客户端列名。
func (s *Service) List(ctx context.Context, q Query) (response.Page[View], error) {
	if err := q.Normalize(); err != nil {
		return response.Page[View]{}, err
	}
	column, desc, err := q.Sort(map[string]string{"id": "id", "sort": "sort", "name": "name"})
	if err != nil {
		return response.Page[View]{}, err
	}
	rows, total, err := s.repo.list(ctx, q, column, desc)
	if err != nil {
		return response.Page[View]{}, err
	}
	out := response.Page[View]{List: []View{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, row := range rows {
		out.List = append(out.List, view(row))
	}
	return out, nil
}

// Get 返回指定菜单。
func (s *Service) Get(ctx context.Context, id model.ID) (View, error) {
	row, err := s.repo.get(ctx, nil, id)
	return view(row), httpx.DBError(err)
}

// Tree 返回含按钮的完整树。
func (s *Service) Tree(ctx context.Context) ([]View, error) {
	rows, err := s.repo.all(ctx, nil)
	if err != nil {
		return nil, err
	}
	return buildTree(rows, 0), nil
}
func buildTree(rows []menu, parent model.ID) []View {
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

var permissionPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*:[a-z][a-z0-9_-]*:[a-zA-Z][a-zA-Z0-9_-]*$`)

// Save 防止父级环与按钮作为父级，权限标识不接受通配符。
func (s *Service) Save(ctx context.Context, p security.Principal, id model.ID, in Input) (View, error) {
	var out menu
	err := s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		if in.ParentID < 0 || strings.TrimSpace(in.Name) == "" || in.Type < 1 || in.Type > 3 || in.Status < 1 || in.Status > 2 || in.Visible < 1 || in.Visible > 2 {
			return nil, response.ErrInvalidParam
		}
		if in.Perm != nil && *in.Perm != "" && !permissionPattern.MatchString(*in.Perm) {
			return nil, response.ErrInvalidParam
		}
		rows, err := s.repo.all(ctx, tx)
		if err != nil {
			return nil, err
		}
		if id == 0 {
			n, err := s.ids.Next()
			if err != nil {
				return nil, err
			}
			out = menu{ID: model.ID(n), CreatedBy: p.ID}
		} else {
			out, err = s.repo.get(ctx, tx, id)
			if err != nil {
				return nil, httpx.DBError(err)
			}
		}
		byID := map[model.ID]menu{}
		for _, row := range rows {
			byID[row.ID] = row
		}
		seen := map[model.ID]bool{out.ID: true}
		for parent := in.ParentID; parent != 0; {
			row, exists := byID[parent]
			if !exists {
				return nil, response.ErrNotFound
			}
			if seen[parent] || row.Type == 3 {
				return nil, response.ErrConflict
			}
			seen[parent] = true
			parent = row.ParentID
		}
		if in.Type == 3 {
			for _, row := range rows {
				if row.ParentID == out.ID {
					return nil, response.ErrConflict
				}
			}
			if in.Perm == nil || *in.Perm == "" {
				return nil, response.ErrInvalidParam
			}
		}
		out.ParentID = in.ParentID
		out.Name = strings.TrimSpace(in.Name)
		out.Type = in.Type
		out.Path = in.Path
		out.Component = in.Component
		out.Perm = in.Perm
		out.Icon = in.Icon
		out.Sort = in.Sort
		out.Visible = in.Visible
		out.Status = in.Status
		out.Remark = in.Remark
		out.UpdatedBy = p.ID
		return nil, httpx.DBError(s.repo.save(ctx, tx, &out, id == 0))
	})
	return view(out), err
}

// Remove 有子节点时拒绝，关联授权在同一事务物理解绑。
func (s *Service) Remove(ctx context.Context, p security.Principal, ids []model.ID) error {
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return response.ErrInvalidParam
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		rows, err := s.repo.all(ctx, tx)
		if err != nil {
			return nil, err
		}
		targets := map[model.ID]bool{}
		for _, id := range ids {
			targets[id] = true
			if _, err := s.repo.get(ctx, tx, id); err != nil {
				return nil, httpx.DBError(err)
			}
		}
		for _, row := range rows {
			if targets[row.ParentID] {
				return nil, response.ErrConflict
			}
		}
		if err := s.roles.RemoveMenus(ctx, tx, ids); err != nil {
			return nil, err
		}
		return nil, s.repo.remove(ctx, tx, ids, p.ID)
	})
}

// Permissions 排除停用、删除及祖级停用的节点，供策略投影使用。
func (s *Service) Permissions(ctx context.Context, tx *gorm.DB) (map[model.ID]string, error) {
	rows, err := s.repo.all(ctx, tx)
	if err != nil {
		return nil, err
	}
	out := map[model.ID]string{}
	byID := map[model.ID]menu{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	for _, row := range rows {
		valid := row.Status == 1
		seen := map[model.ID]bool{row.ID: true}
		for parent := row.ParentID; valid && parent != 0; {
			v, ok := byID[parent]
			if !ok || v.Status != 1 || seen[parent] {
				valid = false
				break
			}
			seen[parent] = true
			parent = v.ParentID
		}
		if valid && row.Perm != nil && *row.Perm != "" {
			out[row.ID] = *row.Perm
		}
	}
	return out, nil
}

// Validate 校验菜单授权只能引用现有且启用的节点。
func (s *Service) Validate(ctx context.Context, tx *gorm.DB, ids []model.ID) error {
	for _, id := range ids {
		row, err := s.repo.get(ctx, tx, id)
		if err != nil {
			return httpx.DBError(err)
		}
		if row.Status != 1 {
			return response.ErrConflict
		}
	}
	return nil
}
