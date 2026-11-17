package dept

import (
	"context"
	"strconv"
	"strings"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

// SetScopeResolver 接入业务范围解析。
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
func checkDepartment(ctx context.Context, id model.ID) error {
	if scope, ok := security.CurrentScope(ctx); ok && !scope.AllowsDepartment(id) {
		return response.ErrPermission
	}
	return nil
}
func departmentScope(ctx context.Context, db *gorm.DB) *gorm.DB {
	if scope, ok := security.CurrentScope(ctx); ok {
		return scope.Apply(db, "id", "", true)
	}
	return db
}

// Descendants 只返回有效部门，自身及祖级匹配按逗号边界判断。
func (s *Service) Descendants(ctx context.Context, tx *gorm.DB, id model.ID) ([]model.ID, error) {
	rows, err := s.repo.all(ctx, tx)
	if err != nil {
		return nil, err
	}
	out := []model.ID{}
	exists := false
	for _, r := range rows {
		if r.ID == id && r.Status == 1 {
			exists = true
		}
		if r.Status == 1 && (r.ID == id || strings.Contains(","+r.Ancestors+",", ","+strconv.FormatInt(int64(id), 10)+",")) {
			out = append(out, r.ID)
		}
	}
	if !exists {
		return []model.ID{}, nil
	}
	return out, nil
}
func scopedTree(ctx context.Context, rows []department) []View {
	scope, ok := security.CurrentScope(ctx)
	if !ok || scope.All {
		return buildTree(rows, 0)
	}
	allowed := map[model.ID]bool{}
	for _, r := range rows {
		if scope.AllowsDepartment(r.ID) || (scope.Self && scope.DepartmentID != nil && r.ID == *scope.DepartmentID) {
			allowed[r.ID] = true
		}
	}
	filtered := []department{}
	for _, r := range rows {
		if allowed[r.ID] {
			if !allowed[r.ParentID] {
				r.ParentID = 0
			}
			filtered = append(filtered, r)
		}
	}
	return buildTree(filtered, 0)
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

// Tree 在安全读锁内构造当前可见树。
func (s *Service) Tree(ctx context.Context) ([]View, error) {
	var out []View
	err := s.read(func() error { var err error; out, err = s.tree(ctx); return err })
	return out, err
}
