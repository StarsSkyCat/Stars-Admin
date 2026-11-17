package security

import (
	"context"
	"slices"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"gorm.io/gorm"
)

// Scope 是业务服务解析完成的可信范围，多角色取并集，零值拒绝全部。
type Scope struct {
	All          bool
	Departments  []model.ID
	Self         bool
	UserID       model.ID
	DepartmentID *model.ID
}

// ScopeRule 不包含角色数据库模型。
type ScopeRule struct {
	Kind        int
	Departments []model.ID
}

// Resolver 由模块装配注入，核心层不查询业务表。
type Resolver interface {
	Resolve(context.Context, *gorm.DB, Principal) (Scope, error)
}
type principalKey struct{}
type scopeKey struct{}

// WithPrincipal 将已认证主体放入请求上下文。
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// Actor 返回可信请求主体；缺失时返回零主体。
func Actor(ctx context.Context) Principal { p, _ := ctx.Value(principalKey{}).(Principal); return p }

// WithScope 仅供服务层将解析结果传递至仓储。
func WithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, s)
}

// CurrentScope 返回服务解析的范围及是否已设置。
func CurrentScope(ctx context.Context) (Scope, bool) {
	s, ok := ctx.Value(scopeKey{}).(Scope)
	return s, ok
}

// AllowsDepartment 校验目标部门，用于写入及授权目标检查。
func (s Scope) AllowsDepartment(id model.ID) bool { return s.All || slices.Contains(s.Departments, id) }

// Apply 将范围映射到资源列；缺少资源映射时默认拒绝。
func (s Scope) Apply(db *gorm.DB, departmentColumn, ownerColumn string, departmentResource bool) *gorm.DB {
	if departmentColumn == "" && ownerColumn == "" {
		return db.Where("1 = 0")
	}
	if s.All {
		return db
	}
	condition := db.Session(&gorm.Session{NewDB: true}).Where("1 = 0")
	if departmentColumn != "" && len(s.Departments) > 0 {
		condition = condition.Or(departmentColumn+" IN ?", s.Departments)
	}
	if s.Self && s.UserID > 0 {
		if departmentResource {
			if s.DepartmentID != nil {
				condition = condition.Or("id = ?", *s.DepartmentID)
			}
		} else if ownerColumn != "" {
			condition = condition.Or(ownerColumn+" = ?", s.UserID)
		}
	}
	return db.Where(condition)
}
