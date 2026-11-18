package security

import (
	"context"
	"errors"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"gorm.io/gorm"
)

func TestScopeDefaultsAndDepartmentUnion(t *testing.T) {
	if (Scope{}).AllowsDepartment(10) {
		t.Fatal("空范围必须拒绝")
	}
	scope := Scope{Departments: []model.ID{10, 11, 20}, Self: true, UserID: 2}
	if !scope.AllowsDepartment(11) || scope.AllowsDepartment(100) {
		t.Fatal("部门集合或路径边界不正确")
	}
	if !(Scope{All: true}).AllowsDepartment(999) {
		t.Fatal("全部范围失效")
	}
	if Current, _ := CurrentScope(t.Context()); Current.All {
		t.Fatal("无可信上下文不能获得全部范围")
	}
}

type scopeTestGate struct {
	reading bool
	denied  bool
}

func (g *scopeTestGate) Read(run func() error) error {
	g.reading = true
	defer func() { g.reading = false }()
	return run()
}

func (g *scopeTestGate) Allow(Principal, string) error {
	if !g.reading || g.denied {
		return response.ErrPermission
	}
	return nil
}

func (g *scopeTestGate) Resolve(context.Context, *gorm.DB, Principal) (Scope, error) {
	if !g.reading {
		return Scope{}, response.ErrPermission
	}
	return Scope{Departments: []model.ID{10}}, nil
}

func TestScopeAccessChecksPermissionInsideReadGate(t *testing.T) {
	gate := new(scopeTestGate)
	access := NewScopeAccess(gate, gate)
	p := Principal{ID: 2}
	query := func(ctx context.Context) (int, error) {
		scope, _ := CurrentScope(ctx)
		if !gate.reading || !scope.AllowsDepartment(10) {
			t.Fatal("查询未在安全门禁或可信范围内执行")
		}
		return 1, nil
	}
	if count, err := access.Read(t.Context(), p, "sys:user:export", query); count != 1 || err != nil {
		t.Fatalf("已授权查询失败：%d，%v", count, err)
	}
	gate.denied = true
	for _, candidate := range []ScopeAccess{access, {}} {
		_, err := candidate.Read(t.Context(), p, "sys:user:export", func(context.Context) (int, error) {
			t.Fatal("权限撤销或装配缺失后执行了查询")
			return 0, nil
		})
		if !errors.Is(err, response.ErrPermission) {
			t.Fatalf("拒绝错误不正确：%v", err)
		}
	}
}
