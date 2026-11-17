package role

import (
	"context"
	"errors"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

type deniedGate struct{ writes int }

func (g *deniedGate) Change(context.Context, func(*gorm.DB) ([]security.Revocation, error)) error {
	g.writes++
	return errors.New("不应写入")
}
func (*deniedGate) Allow(security.Principal, string) error { return response.ErrPermission }

func TestRoleReservedAndGrantGuards(t *testing.T) {
	g := new(deniedGate)
	s := &Service{gate: g}
	admin := security.Principal{ID: 1, RoleIDs: []model.ID{1}}
	for _, fn := range []func() error{
		func() error { return s.Status(t.Context(), admin, 1, 2) }, func() error { return s.Remove(t.Context(), admin, []model.ID{2, 1}) }, func() error { return s.GrantMenus(t.Context(), security.Principal{}, 2, []model.ID{3}) }, func() error { return s.GrantScope(t.Context(), security.Principal{}, 2, ScopeInput{DataScope: 1}) },
	} {
		if err := fn(); !errors.Is(err, response.ErrPermission) {
			t.Fatal("保留角色或危险授权未拒绝")
		}
	}
	if g.writes != 0 {
		t.Fatal("保护校验之前开始了写入")
	}
}
