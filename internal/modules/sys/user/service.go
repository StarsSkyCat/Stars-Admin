package user

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/permission"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/StarsSkyCat/Stars-Admin/pkg/hash"
	"gorm.io/gorm"
)

type idSource interface{ Next() (int64, error) }
type changeGate interface {
	Change(context.Context, func(*gorm.DB) ([]security.Revocation, error)) error
	Update(context.Context, func(*gorm.DB) ([]security.Revocation, error)) error
	Allow(security.Principal, string) error
}

// Roles 是用户业务消费的角色能力，不共享角色数据库模型。
type Roles interface {
	Resolve(context.Context, *gorm.DB, []model.ID) ([]model.ID, error)
	CanAssign(context.Context, *gorm.DB, security.Principal, model.ID, []model.ID) error
	Permissions(context.Context, *gorm.DB, []model.ID) ([]string, error)
}

// Departments 校验有效部门引用。
type Departments interface {
	Exists(context.Context, *gorm.DB, model.ID) error
}

// Service 提供用户管理、认证快照与同事务撤销能力。
type Service struct {
	repo   repository
	ids    idSource
	gate   changeGate
	roles  Roles
	depts  Departments
	cost   int
	access security.ScopeAccess
	posts  Posts
}

// New 创建用户业务，哈希成本由应用配置决定。
func New(db *gorm.DB, ids idSource, gate changeGate, roles Roles, depts Departments, access security.ScopeAccess, cost int) *Service {
	return &Service{repo: repository{db}, ids: ids, gate: gate, roles: roles, depts: depts, access: access, cost: cost}
}

// list 查询已授权范围内的分页资料。
func (s *Service) list(ctx context.Context, q Query) (response.Page[View], error) {
	if err := q.Normalize(); err != nil {
		return response.Page[View]{}, err
	}
	rows, count, err := s.repo.list(ctx, q)
	out := response.Page[View]{List: []View{}, Total: count, Page: q.Page, PageSize: q.PageSize}
	for _, row := range rows {
		out.List = append(out.List, view(row))
	}
	return out, err
}

// get 查询已授权范围内的用户详情及关联。
func (s *Service) get(ctx context.Context, id model.ID) (View, error) {
	row, err := s.repo.get(ctx, nil, id)
	if err != nil {
		return View{}, httpx.DBError(err)
	}
	out := view(row)
	ids, err := s.repo.roles(ctx, nil, id)
	out.RoleIDs = ids
	if err != nil {
		return out, err
	}
	out.PostIDs, err = s.repo.posts(ctx, nil, id)
	return out, err
}

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.@-]{0,63}$`)

// Create 新用户默认启用、无角色且强制首次改密。
func (s *Service) Create(ctx context.Context, p security.Principal, in CreateInput) (View, error) {
	encoded, err := hash.Password(in.Password, s.cost)
	if err != nil {
		return View{}, response.ErrInvalidParam
	}
	return s.save(ctx, p, 0, in.Input, encoded)
}

// Save 通用编辑不接受密码、状态和角色。
func (s *Service) Save(ctx context.Context, p security.Principal, id model.ID, in Input) (View, error) {
	return s.save(ctx, p, id, in, "")
}
func (s *Service) save(ctx context.Context, p security.Principal, id model.ID, in Input, encoded string) (View, error) {
	if id == 1 && !p.Admin() {
		return View{}, response.ErrPermission
	}
	var out user
	err := s.gate.Update(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		perm := "sys:user:edit"
		if id == 0 {
			perm = "sys:user:add"
		}
		unscoped := ctx
		ctx, scopeErr := s.access.Context(ctx, tx, p, perm)
		if scopeErr != nil {
			return nil, scopeErr
		}
		in.Username = strings.ToLower(strings.TrimSpace(in.Username))
		if !usernamePattern.MatchString(in.Username) || strings.TrimSpace(in.Nickname) == "" || in.Gender < 0 || in.Gender > 2 {
			return nil, response.ErrInvalidParam
		}
		if in.DeptID != nil {
			if err := s.checkDepartment(ctx, *in.DeptID); err != nil {
				scope, _ := security.CurrentScope(ctx)
				if id != p.ID || !scope.Self || scope.DepartmentID == nil || *scope.DepartmentID != *in.DeptID {
					return nil, err
				}
			}
			if *in.DeptID <= 0 {
				return nil, response.ErrInvalidParam
			}
			if err := s.depts.Exists(ctx, tx, *in.DeptID); err != nil {
				return nil, err
			}
		}

		if id == 0 {
			scope, _ := security.CurrentScope(ctx)
			if !scope.All && in.DeptID == nil {
				return nil, response.ErrPermission
			}
		}
		if id == 0 {
			if in.Username == "admin" {
				return nil, response.ErrPermission
			}
			n, err := s.ids.Next()
			if err != nil {
				return nil, err
			}
			out = user{ID: model.ID(n), CreatedBy: p.ID, Password: encoded, AuthVersion: 1, Status: 1}
		} else {
			var err error
			out, err = s.repo.get(ctx, tx, id)
			if err != nil {
				return nil, httpx.DBError(err)
			}

			if (id == 1 && in.Username != "admin") || (id != 1 && in.Username == "admin") {
				return nil, response.ErrPermission
			}
		}
		if in.DeptID == nil {
			if scope, ok := security.CurrentScope(ctx); ok && !scope.All && (!scope.Self || out.ID != p.ID) {
				return nil, response.ErrPermission
			}
		}
		scope, _ := security.CurrentScope(ctx)
		movingSelf := id == p.ID && !scope.All && in.DeptID != nil && (out.DeptID == nil || *out.DeptID != *in.DeptID)
		out.Username = in.Username
		out.Nickname = strings.TrimSpace(in.Nickname)
		out.Email = in.Email
		out.Phone = in.Phone
		out.Gender = in.Gender
		out.Avatar = in.Avatar
		out.DeptID = in.DeptID
		out.Remark = in.Remark
		out.UpdatedBy = p.ID
		if err := s.repo.save(ctx, tx, &out, id == 0); err != nil {
			return nil, httpx.DBError(err)
		}
		if movingSelf {
			// 按事务内的新部门重新计算，任何新增可见部门都要求管理员授权。
			updated, err := s.access.Context(unscoped, tx, p, perm)
			if err != nil {
				return nil, err
			}
			next, _ := security.CurrentScope(updated)
			if next.All || slices.ContainsFunc(next.Departments, func(dept model.ID) bool {
				return !scope.AllowsDepartment(dept)
			}) {
				return nil, response.ErrPermission
			}
		}
		if id != 0 {
			return s.repo.bump(ctx, tx, []model.ID{id})
		}
		return nil, nil
	})
	return view(out), err
}

// Status 对停用和恢复操作均提升会话版本。
func (s *Service) Status(ctx context.Context, p security.Principal, id model.ID, status int) error {
	if status != 1 && status != 2 {
		return response.ErrInvalidParam
	}
	if id == 1 {
		return response.ErrPermission
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		var scopeErr error
		ctx, scopeErr = s.access.Context(ctx, tx, p, "sys:user:edit")
		if scopeErr != nil {
			return nil, scopeErr
		}
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
		return s.repo.bump(ctx, tx, []model.ID{id})
	})
}

// ResetPassword 重置后清空改密时间，撤销所有 Access 与 Refresh。
func (s *Service) ResetPassword(ctx context.Context, p security.Principal, id model.ID, password string) error {
	if id == 1 && !p.Admin() {
		return response.ErrPermission
	}
	if err := s.gate.Allow(p, "sys:user:resetPwd"); err != nil {
		return err
	}
	return s.password(ctx, p, id, "", password, true)
}

// ChangePassword 必须核对旧密码，成功后本人所有会话失效。
func (s *Service) ChangePassword(ctx context.Context, p security.Principal, old, password string) error {
	return s.password(ctx, p, p.ID, old, password, false)
}
func (s *Service) password(ctx context.Context, p security.Principal, id model.ID, old, password string, reset bool) error {
	encoded, err := hash.Password(password, s.cost)
	if err != nil {
		return response.ErrInvalidParam
	}
	return s.gate.Update(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		var scopeErr error
		if reset {
			ctx, scopeErr = s.access.Context(ctx, tx, p, "sys:user:resetPwd")
		} else {
			ctx = security.WithScope(ctx, security.Scope{Self: true, UserID: p.ID})
		}
		if scopeErr != nil {
			return nil, scopeErr
		}
		row, err := s.repo.get(ctx, tx, id)
		if err != nil {
			return nil, httpx.DBError(err)
		}
		if !reset {
			if row.AuthVersion != p.AuthVersion {
				return nil, response.ErrUnauthenticated
			}
			if !hash.Match(row.Password, old) {
				return nil, response.ErrCredentials
			}
			if hash.Match(row.Password, password) {
				return nil, response.ErrInvalidParam
			}
		}
		row.Password = encoded
		row.UpdatedBy = p.ID
		row.PwdUpdatedAt = nil
		if !reset {
			row.PwdUpdatedAt = new(time.Now().UTC())
		}
		if err := s.repo.save(ctx, tx, &row, false); err != nil {
			return nil, err
		}
		return s.repo.bump(ctx, tx, []model.ID{id})
	})
}

// GrantRoles 保留账号不能失去 admin，其他账号不能取得保留角色。
func (s *Service) GrantRoles(ctx context.Context, p security.Principal, id model.ID, ids []model.ID) error {
	if id == 1 && !p.Admin() {
		return response.ErrPermission
	}
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	if err := s.gate.Allow(p, "sys:user:grant"); err != nil {
		return err
	}
	if id == 1 && !slices.Contains(ids, model.ID(1)) {
		return response.ErrPermission
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		var scopeErr error
		ctx, scopeErr = s.access.Context(ctx, tx, p, "sys:user:grant")
		if scopeErr != nil {
			return nil, scopeErr
		}
		if _, err := s.repo.get(ctx, tx, id); err != nil {
			return nil, httpx.DBError(err)
		}
		if err := s.roles.CanAssign(ctx, tx, p, id, ids); err != nil {
			return nil, err
		}
		rows := []userRole{}
		for _, roleID := range ids {
			n, err := s.ids.Next()
			if err != nil {
				return nil, err
			}
			rows = append(rows, userRole{ID: model.ID(n), CreatedBy: p.ID, UpdatedBy: p.ID, UserID: id, RoleID: roleID})
		}
		if err := s.repo.replaceRoles(ctx, tx, id, rows); err != nil {
			return nil, err
		}
		return s.repo.bump(ctx, tx, []model.ID{id})
	})
}

// Remove 批量整体校验保留 ID 并先撤销版本再软删除。
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
		var scopeErr error
		ctx, scopeErr = s.access.Context(ctx, tx, p, "sys:user:remove")
		if scopeErr != nil {
			return nil, scopeErr
		}
		for _, id := range ids {
			if _, err := s.repo.get(ctx, tx, id); err != nil {
				return nil, httpx.DBError(err)
			}
		}
		revoked, err := s.repo.bump(ctx, tx, ids)
		if err != nil {
			return nil, err
		}
		return revoked, s.repo.remove(ctx, tx, ids, p.ID)
	})
}

func (s *Service) account(ctx context.Context, row user) (security.Account, error) {
	ids, err := s.repo.roles(ctx, nil, row.ID)
	if err != nil {
		return security.Account{}, err
	}
	active, err := s.roles.Resolve(ctx, nil, ids)
	return security.Account{Principal: security.Principal{ID: row.ID, Username: row.Username, RoleIDs: active, AuthVersion: row.AuthVersion, MustChangePassword: row.PwdUpdatedAt == nil}, Password: row.Password, Nickname: row.Nickname, Status: row.Status}, err
}

// ByUsername 只供认证服务读取哈希快照。
func (s *Service) ByUsername(ctx context.Context, name string) (security.Account, error) {
	row, err := s.repo.byName(ctx, name)
	if err != nil {
		return security.Account{}, httpx.DBError(err)
	}
	return s.account(ctx, row)
}

// Account 每次请求重新读取持久状态，避免陈旧会话恢复。
func (s *Service) Account(ctx context.Context, id model.ID) (security.Account, error) {
	row, err := s.repo.get(ctx, nil, id)
	if err != nil {
		return security.Account{}, httpx.DBError(err)
	}
	return s.account(ctx, row)
}

// Permissions 只读取当前有效角色的权限。
func (s *Service) Permissions(ctx context.Context, ids []model.ID) ([]string, error) {
	return s.roles.Permissions(ctx, nil, ids)
}

// RecordLogin 更新最后登录元信息，不延长会话有效期。
func (s *Service) RecordLogin(ctx context.Context, id model.ID, ip string) error {
	return s.repo.login(ctx, id, ip)
}

// HasDepartments 提供部门删除保护。
func (s *Service) HasDepartments(ctx context.Context, tx *gorm.DB, ids []model.ID) (bool, error) {
	return s.repo.hasDepartments(ctx, tx, ids)
}

// HasRoles 提供角色删除保护。
func (s *Service) HasRoles(ctx context.Context, tx *gorm.DB, ids []model.ID) (bool, error) {
	return s.repo.hasRoles(ctx, tx, ids)
}

// RevokeRoles 在调用方事务内提升所有角色成员版本。
func (s *Service) RevokeRoles(ctx context.Context, tx *gorm.DB, ids []model.ID) ([]security.Revocation, error) {
	users, err := s.repo.roleUsers(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	return s.repo.bump(ctx, tx, users)
}

// Memberships 提供仅属于有效用户的授权边。
func (s *Service) Memberships(ctx context.Context, tx *gorm.DB) ([]permission.Membership, error) {
	rows, err := s.repo.members(ctx, tx)
	if err != nil {
		return nil, err
	}
	out := make([]permission.Membership, 0, len(rows))
	for _, row := range rows {
		out = append(out, permission.Membership{UserID: int64(row.UserID), RoleID: int64(row.RoleID)})
	}
	return out, nil
}
