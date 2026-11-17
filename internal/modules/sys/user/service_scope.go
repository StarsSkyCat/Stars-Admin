package user

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

// SetScopeResolver 在装配时接入可信范围解析。
func (s *Service) SetScopeResolver(r security.Resolver) { s.scopes = r }
func (s *Service) scopeContext(ctx context.Context, tx *gorm.DB, p security.Principal, permissions ...string) (context.Context, error) {
	if s.scopes == nil {
		return ctx, nil
	}
	scope, err := s.scopes.Resolve(ctx, tx, p)
	if err == nil && tx != nil && len(permissions) > 0 {
		if checker, ok := s.gate.(interface {
			Allow(security.Principal, string) error
		}); ok {
			err = checker.Allow(p, permissions[0])
		}
	}
	return security.WithScope(ctx, scope), err
}
func (s *Service) checkDepartment(ctx context.Context, id model.ID) error {
	scope, ok := security.CurrentScope(ctx)
	if ok && !scope.AllowsDepartment(id) {
		return response.ErrPermission
	}
	return nil
}

// DepartmentID 为范围解析提供当前用户的部门，不应用资源过滤。
func (s *Service) DepartmentID(ctx context.Context, tx *gorm.DB, id model.ID) (*model.ID, error) {
	return s.repo.department(ctx, tx, id)
}

// ScopeIdentity 在事务内核对当前版本与角色成员，不信任先前请求快照。
func (s *Service) ScopeIdentity(ctx context.Context, tx *gorm.DB, p security.Principal) (security.Principal, error) {
	row, err := s.repo.identity(ctx, tx, p.ID)
	if err != nil {
		return security.Principal{}, httpx.DBError(err)
	}
	if row.Status != 1 || row.AuthVersion != p.AuthVersion {
		return security.Principal{}, response.ErrUnauthenticated
	}
	ids, err := s.repo.roles(ctx, tx, p.ID)
	if err != nil {
		return security.Principal{}, err
	}
	active, err := s.roles.Resolve(ctx, tx, ids)
	if err != nil {
		return security.Principal{}, err
	}
	if len(active) == 0 {
		return security.Principal{}, response.ErrPermission
	}
	return security.Principal{ID: row.ID, Username: row.Username, RoleIDs: active, AuthVersion: row.AuthVersion, MustChangePassword: row.PwdUpdatedAt == nil}, nil
}

func (s *Service) read(fn func() error) error {
	if reader, ok := s.gate.(interface{ Read(func() error) error }); ok {
		return reader.Read(fn)
	}
	return fn()
}

// List 在安全读锁内解析范围并查询，避免与范围收缩交错。
func (s *Service) List(ctx context.Context, q Query) (response.Page[View], error) {
	var out response.Page[View]
	err := s.read(func() error { var err error; out, err = s.list(ctx, q); return err })
	return out, err
}

// Get 在同一次安全读取中完成对象范围检查。
func (s *Service) Get(ctx context.Context, id model.ID) (View, error) {
	var out View
	err := s.read(func() error { var err error; out, err = s.get(ctx, id); return err })
	return out, err
}
