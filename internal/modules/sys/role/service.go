package role

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/permission"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

type idSource interface{ Next() (int64, error) }
type changeGate interface {
	Change(context.Context, func(*gorm.DB) ([]security.Revocation, error)) error
	Allow(security.Principal, string) error
}

// Menus 是角色业务需要的菜单能力。
type Menus interface {
	Validate(context.Context, *gorm.DB, []model.ID) error
	Permissions(context.Context, *gorm.DB) (map[model.ID]string, error)
}

// Departments 校验自定义数据范围引用。
type Departments interface {
	Exists(context.Context, *gorm.DB, model.ID) error
}

// Users 提供关联保护和同事务会话版本提升。
type Users interface {
	HasRoles(context.Context, *gorm.DB, []model.ID) (bool, error)
	RevokeRoles(context.Context, *gorm.DB, []model.ID) ([]security.Revocation, error)
}

// Service 管理角色普通编辑与危险授权入口。
type Service struct {
	repo  repository
	ids   idSource
	gate  changeGate
	menus Menus
	depts Departments
	users Users
}

// New 构造角色业务并注入前置能力。
func New(db *gorm.DB, ids idSource, gate changeGate, menus Menus, depts Departments) *Service {
	return &Service{repo: repository{db}, ids: ids, gate: gate, menus: menus, depts: depts}
}

// SetUsers 完成引用保护装配。
func (s *Service) SetUsers(users Users) { s.users = users }

// List 分页查询角色。
func (s *Service) List(ctx context.Context, q Query) (response.Page[View], error) {
	if err := q.Normalize(); err != nil {
		return response.Page[View]{}, err
	}
	rows, total, err := s.repo.list(ctx, q)
	out := response.Page[View]{List: []View{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, row := range rows {
		out.List = append(out.List, view(row))
	}
	return out, err
}

// Get 同时返回授权 ID，用于专用编辑页面。
func (s *Service) Get(ctx context.Context, id model.ID) (View, error) {
	row, err := s.repo.get(ctx, nil, id)
	if err != nil {
		return View{}, httpx.DBError(err)
	}
	out := view(row)
	menus, err := s.repo.menus(ctx, nil, []model.ID{id})
	if err != nil {
		return View{}, err
	}
	for _, m := range menus {
		out.MenuIDs = append(out.MenuIDs, m.MenuID)
	}
	depts, err := s.repo.depts(ctx, nil, id)
	if err != nil {
		return View{}, err
	}
	out.DeptIDs = append(out.DeptIDs, depts...)
	return out, nil
}

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// Save 新角色默认启用且仅本人范围，普通编辑不接受授权字段。
func (s *Service) Save(ctx context.Context, p security.Principal, id model.ID, in Input) (View, error) {
	var out role
	err := s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		in.Code = strings.ToLower(strings.TrimSpace(in.Code))
		if !codePattern.MatchString(in.Code) || strings.TrimSpace(in.Name) == "" {
			return nil, response.ErrInvalidParam
		}
		if id == 0 {
			if in.Code == "admin" {
				return nil, response.ErrPermission
			}
			n, err := s.ids.Next()
			if err != nil {
				return nil, err
			}
			out = role{ID: model.ID(n), CreatedBy: p.ID, Status: 1, DataScope: 5}
		} else {
			var err error
			out, err = s.repo.get(ctx, tx, id)
			if err != nil {
				return nil, httpx.DBError(err)
			}
			if (id == 1 && in.Code != "admin") || (id != 1 && in.Code == "admin") {
				return nil, response.ErrPermission
			}
		}
		out.Name = strings.TrimSpace(in.Name)
		out.Code = in.Code
		out.Sort = in.Sort
		out.Remark = in.Remark
		out.UpdatedBy = p.ID
		return nil, httpx.DBError(s.repo.save(ctx, tx, &out, id == 0))
	})
	return view(out), err
}

// Status 角色状态变化撤销所有关联用户的会话。
func (s *Service) Status(ctx context.Context, p security.Principal, id model.ID, status int) error {
	if status != 1 && status != 2 {
		return response.ErrInvalidParam
	}
	if id == 1 {
		return response.ErrPermission
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		row, err := s.repo.get(ctx, tx, id)
		if err != nil {
			return nil, httpx.DBError(err)
		}
		if row.Status == status {
			return nil, nil
		}
		row.Status = status
		row.UpdatedBy = p.ID
		if err := s.repo.save(ctx, tx, &row, false); err != nil {
			return nil, err
		}
		return s.users.RevokeRoles(ctx, tx, []model.ID{id})
	})
}

// GrantMenus 独立验证 grant 权限和可转授集合。
func (s *Service) GrantMenus(ctx context.Context, p security.Principal, id model.ID, ids []model.ID) error {
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	if err := s.gate.Allow(p, "sys:role:grant"); err != nil {
		return err
	}
	if id == 1 && !p.Admin() {
		return response.ErrPermission
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		if _, err := s.repo.get(ctx, tx, id); err != nil {
			return nil, httpx.DBError(err)
		}
		if err := s.menus.Validate(ctx, tx, ids); err != nil {
			return nil, err
		}
		if !p.Admin() {
			perms, err := s.menus.Permissions(ctx, tx)
			if err != nil {
				return nil, err
			}
			for _, menuID := range ids {
				perm := perms[menuID]
				if perm != "" {
					if err := s.gate.Allow(p, perm); err != nil {
						return nil, err
					}
				}
			}
		}
		rows := make([]roleMenu, 0, len(ids))
		for _, menuID := range ids {
			n, err := s.ids.Next()
			if err != nil {
				return nil, err
			}
			rows = append(rows, roleMenu{ID: model.ID(n), CreatedBy: p.ID, UpdatedBy: p.ID, RoleID: id, MenuID: menuID})
		}
		return nil, s.repo.replaceMenus(ctx, tx, id, rows)
	})
}

// GrantScope 仅 admin 可以配置数据范围，后续读取即时按事实解析。
func (s *Service) GrantScope(ctx context.Context, p security.Principal, id model.ID, in ScopeInput) error {
	if !p.Admin() {
		return response.ErrPermission
	}
	if err := s.gate.Allow(p, "sys:role:grant"); err != nil {
		return err
	}
	if in.DataScope < 1 || in.DataScope > 5 {
		return response.ErrInvalidParam
	}
	if err := httpx.UniqueIDs(in.DeptIDs); err != nil {
		return err
	}
	if (in.DataScope == 2 && len(in.DeptIDs) == 0) || (in.DataScope != 2 && len(in.DeptIDs) > 0) {
		return response.ErrInvalidParam
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		row, err := s.repo.get(ctx, tx, id)
		if err != nil {
			return nil, httpx.DBError(err)
		}
		rows := []roleDept{}
		for _, deptID := range in.DeptIDs {
			if err := s.depts.Exists(ctx, tx, deptID); err != nil {
				return nil, err
			}
			n, err := s.ids.Next()
			if err != nil {
				return nil, err
			}
			rows = append(rows, roleDept{ID: model.ID(n), CreatedBy: p.ID, UpdatedBy: p.ID, RoleID: id, DeptID: deptID})
		}
		row.DataScope = in.DataScope
		row.UpdatedBy = p.ID
		if err := s.repo.save(ctx, tx, &row, false); err != nil {
			return nil, err
		}
		return nil, s.repo.replaceDepts(ctx, tx, id, rows)
	})
}

// Remove 保留角色与有用户引用的角色不能删除。
func (s *Service) Remove(ctx context.Context, p security.Principal, ids []model.ID) error {
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return response.ErrInvalidParam
	}
	if slices.Contains(ids, model.ID(1)) {
		return response.ErrPermission
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		for _, id := range ids {
			if _, err := s.repo.get(ctx, tx, id); err != nil {
				return nil, httpx.DBError(err)
			}
		}
		used, err := s.users.HasRoles(ctx, tx, ids)
		if err != nil {
			return nil, err
		}
		if used {
			return nil, response.ErrConflict
		}
		return nil, s.repo.remove(ctx, tx, ids, p.ID)
	})
}

// Resolve 返回仍启用的角色，并用于登录与每次请求的当前状态核对。
func (s *Service) Resolve(ctx context.Context, tx *gorm.DB, ids []model.ID) ([]model.ID, error) {
	rows, err := s.repo.active(ctx, tx)
	if err != nil {
		return nil, err
	}
	out := []model.ID{}
	for _, row := range rows {
		if slices.Contains(ids, row.ID) {
			out = append(out, row.ID)
		}
	}
	return out, nil
}

// CanAssign 限制普通账号只转授本人持有且仅本人范围的非 admin 角色。
func (s *Service) CanAssign(ctx context.Context, tx *gorm.DB, p security.Principal, target model.ID, ids []model.ID) error {
	for _, id := range ids {
		row, err := s.repo.get(ctx, tx, id)
		if err != nil {
			return httpx.DBError(err)
		}
		if row.Status != 1 {
			return response.ErrConflict
		}
		if id == 1 && target != 1 {
			return response.ErrPermission
		}
		if !p.Admin() && (!slices.Contains(p.RoleIDs, id) || row.DataScope != 5 || id == 1) {
			return response.ErrPermission
		}
	}
	return nil
}

// ActiveIDs 供模块投影装配过滤停用角色。
func (s *Service) ActiveIDs(ctx context.Context, tx *gorm.DB) ([]model.ID, error) {
	rows, err := s.repo.active(ctx, tx)
	out := []model.ID{}
	for _, row := range rows {
		out = append(out, row.ID)
	}
	return out, err
}

// Grants 提供与数据库模型解耦的授权边。
func (s *Service) Grants(ctx context.Context, tx *gorm.DB) ([]permission.Grant, error) {
	rows, err := s.repo.menus(ctx, tx, nil)
	out := []permission.Grant{}
	for _, row := range rows {
		out = append(out, permission.Grant{RoleID: int64(row.RoleID), MenuID: int64(row.MenuID)})
	}
	return out, err
}

// Permissions 返回指定有效角色的当前权限集合。
func (s *Service) Permissions(ctx context.Context, tx *gorm.DB, ids []model.ID) ([]string, error) {
	perms, err := s.menus.Permissions(ctx, tx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.menus(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, row := range rows {
		if perm := perms[row.MenuID]; perm != "" && !slices.Contains(out, perm) {
			out = append(out, perm)
		}
	}
	slices.Sort(out)
	return out, nil
}

// RemoveMenus 在菜单 service 的事务中解绑，不独立提交。
func (s *Service) RemoveMenus(ctx context.Context, tx *gorm.DB, ids []model.ID) error {
	return s.repo.removeMenus(ctx, tx, ids)
}

// ScopeRules 读取有效角色及其仍有效的自定义部门，供模块解析器合并。
func (s *Service) ScopeRules(ctx context.Context, tx *gorm.DB, ids []model.ID) ([]security.ScopeRule, error) {
	rows, err := s.repo.active(ctx, tx)
	if err != nil {
		return nil, err
	}
	out := []security.ScopeRule{}
	for _, r := range rows {
		if !slices.Contains(ids, r.ID) {
			continue
		}
		rule := security.ScopeRule{Kind: r.DataScope}
		if r.DataScope == 2 {
			depts, err := s.repo.depts(ctx, tx, r.ID)
			if err != nil {
				return nil, err
			}
			for _, id := range depts {
				if err := s.depts.Exists(ctx, tx, id); err != nil {
					if errors.Is(err, response.ErrNotFound) || errors.Is(err, response.ErrConflict) {
						continue
					}
					return nil, err
				}
				rule.Departments = append(rule.Departments, id)
			}
		}
		out = append(out, rule)
	}
	return out, nil
}
