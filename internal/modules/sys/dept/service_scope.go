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

// List 在安全读锁内解析范围并查询，避免与范围收缩交错。
func (s *Service) List(ctx context.Context, q Query) (response.Page[View], error) {
	return s.access.Read(ctx, security.Actor(ctx), "sys:dept:list", func(scoped context.Context) (response.Page[View], error) {
		return s.list(scoped, q)
	})
}

// Get 在同一次安全读取中完成对象范围检查。
func (s *Service) Get(ctx context.Context, id model.ID) (View, error) {
	return s.access.Read(ctx, security.Actor(ctx), "sys:dept:get", func(scoped context.Context) (View, error) {
		return s.get(scoped, id)
	})
}

// Tree 在安全读锁内构造当前可见树。
func (s *Service) Tree(ctx context.Context) ([]View, error) {
	return s.access.Read(ctx, security.Actor(ctx), "sys:dept:list", s.tree)
}
