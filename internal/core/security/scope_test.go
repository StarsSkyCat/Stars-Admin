package security

import (
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
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
