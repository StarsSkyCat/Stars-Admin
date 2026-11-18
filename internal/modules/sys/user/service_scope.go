package user

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

func (s *Service) checkDepartment(ctx context.Context, id model.ID) error {
	scope, ok := security.CurrentScope(ctx)
	if !ok || !scope.AllowsDepartment(id) {
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

// List 在安全读锁内解析范围并查询，避免与范围收缩交错。
func (s *Service) List(ctx context.Context, q Query) (response.Page[View], error) {
	return s.access.Read(ctx, security.Actor(ctx), "sys:user:list", func(scoped context.Context) (response.Page[View], error) {
		return s.list(scoped, q)
	})
}

// Get 在同一次安全读取中完成对象范围检查。
func (s *Service) Get(ctx context.Context, id model.ID) (View, error) {
	return s.access.Read(ctx, security.Actor(ctx), "sys:user:get", func(scoped context.Context) (View, error) {
		return s.get(scoped, id)
	})
}
