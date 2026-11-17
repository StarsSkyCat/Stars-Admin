//go:build integration

package role

import (
	"context"
	"errors"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/StarsSkyCat/Stars-Admin/internal/testutil"
	"gorm.io/gorm"
)

type grantGate struct {
	db      *gorm.DB
	allowed map[string]bool
}

func (g grantGate) Change(ctx context.Context, fn func(*gorm.DB) ([]security.Revocation, error)) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { _, err := fn(tx); return err })
}
func (g grantGate) Allow(_ security.Principal, perm string) error {
	if !g.allowed[perm] {
		return response.ErrPermission
	}
	return nil
}

type grantMenus struct{}

func (grantMenus) Validate(context.Context, *gorm.DB, []model.ID) error { return nil }
func (grantMenus) Permissions(context.Context, *gorm.DB) (map[model.ID]string, error) {
	return map[model.ID]string{9901: "sys:user:list", 9902: "sys:user:remove"}, nil
}

func TestNonAdminCannotAssignUnheldOrBroaderRoleAndGrantUnheldMenu(t *testing.T) {
	cfg, err := config.Load("../../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	ids := []model.ID{}
	for i, scope := range []int{5, 5, 1} {
		n, err := deps.IDs.Next()
		if err != nil {
			t.Fatal(err)
		}
		row := role{ID: model.ID(n), Name: "转授测试", Code: []string{"m5_held", "m5_unheld", "m5_broad"}[i], DataScope: scope, Status: 1}
		if err := deps.DB.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		ids = append(ids, row.ID)
	}
	p := security.Principal{ID: 8801, RoleIDs: []model.ID{ids[0], ids[2]}}
	g := grantGate{deps.DB, map[string]bool{"sys:role:grant": true, "sys:user:list": true}}
	s := New(deps.DB, deps.IDs, g, grantMenus{}, nil)
	if err := s.CanAssign(t.Context(), nil, p, 8802, []model.ID{ids[0]}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []model.ID{ids[1], ids[2]} {
		if err := s.CanAssign(t.Context(), nil, p, 8802, []model.ID{id}); !errors.Is(err, response.ErrPermission) {
			t.Fatal("未持有或更宽范围的角色被转授")
		}
	}
	if err := s.GrantMenus(t.Context(), p, ids[1], []model.ID{9901, 9902}); !errors.Is(err, response.ErrPermission) {
		t.Fatal("超出自持权限的菜单转授未拒绝")
	}
	var count int64
	if err := deps.DB.Model(&roleMenu{}).Where("role_id = ?", ids[1]).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("转授拒绝未整体回滚")
	}
}
