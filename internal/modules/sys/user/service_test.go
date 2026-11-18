package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

type guardGate struct{ writes int }

func (g *guardGate) Change(context.Context, func(*gorm.DB) ([]security.Revocation, error)) error {
	g.writes++
	return errors.New("不应写入")
}
func (g *guardGate) Update(ctx context.Context, fn func(*gorm.DB) ([]security.Revocation, error)) error {
	return g.Change(ctx, fn)
}
func (*guardGate) Allow(security.Principal, string) error { return nil }

func TestUserReservedAndPasswordGuards(t *testing.T) {
	g := new(guardGate)
	s := &Service{gate: g, cost: 4}
	p := security.Principal{ID: 1, RoleIDs: []model.ID{1}}
	for _, fn := range []func() error{func() error { return s.Status(t.Context(), p, 1, 2) }, func() error { return s.Remove(t.Context(), p, []model.ID{2, 1}) }, func() error { return s.GrantRoles(t.Context(), p, 1, []model.ID{2}) }} {
		if err := fn(); !errors.Is(err, response.ErrPermission) {
			t.Fatal("保留账号保护失效")
		}
	}
	if err := s.ResetPassword(t.Context(), p, 2, strings.Repeat("中", 25)); !errors.Is(err, response.ErrInvalidParam) {
		t.Fatal("bcrypt 字节边界失效")
	}
	if g.writes != 0 {
		t.Fatal("保护校验之前开始写入")
	}
	if err := s.ResetPassword(t.Context(), security.Principal{ID: 2}, 1, "Valid-Password-123"); !errors.Is(err, response.ErrPermission) {
		t.Fatal("普通主体不能重置保留管理员密码")
	}
}

func TestForceLogoutRequiresAdministratorBeforeWrite(t *testing.T) {
	s := new(Service)
	if err := s.RevokeAll(t.Context(), security.Principal{ID: 2}, 3); !errors.Is(err, response.ErrPermission) {
		t.Fatal("无管理身份的强退必须在写入前拒绝")
	}
}
