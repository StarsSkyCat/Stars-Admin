package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
)

type fakeUsers struct {
	account security.Account
	changed bool
}

func (u *fakeUsers) ByUsername(context.Context, string) (security.Account, error) {
	return u.account, nil
}
func (u *fakeUsers) Account(context.Context, model.ID) (security.Account, error) {
	return u.account, nil
}
func (*fakeUsers) Permissions(context.Context, []model.ID) ([]string, error) {
	return []string{"sys:user:list"}, nil
}
func (u *fakeUsers) ChangePassword(_ context.Context, p security.Principal, old, password string) error {
	u.changed = p.ID == 2 && old == "old-password" && password == "new-password-123"
	return nil
}
func (*fakeUsers) RecordLogin(context.Context, model.ID, string) error { return nil }

type openGate struct{}

func (openGate) Read(fn func() error) error             { return fn() }
func (openGate) Allow(security.Principal, string) error { return nil }

func TestProfileAndManagementGate(t *testing.T) {
	users := &fakeUsers{account: security.Account{Principal: security.Principal{ID: 2, Username: "alice", AuthVersion: 1, RoleIDs: []model.ID{3}}, Nickname: "测试", Status: 1}}
	s := &Service{users: users, gate: openGate{}}
	p := users.account.Principal
	profile, err := s.Profile(t.Context(), p)
	if err != nil || profile.ID != 2 || len(profile.Permissions) != 1 {
		t.Fatal("本人权限资料不正确")
	}
	if err := s.Permit(t.Context(), p, "sys:user:list"); !errors.Is(err, response.ErrPermission) {
		t.Fatal("M3 前普通用户不能进入管理接口")
	}
	users.account.MustChangePassword = true
	if err := s.Permit(t.Context(), p, "sys:user:list"); !errors.Is(err, response.ErrPasswordChange) {
		t.Fatal("首登约束未生效")
	}
	users.account.AuthVersion = 2
	if _, err := s.Profile(t.Context(), p); !errors.Is(err, response.ErrUnauthenticated) {
		t.Fatal("过期身份快照不能读取资料")
	}
	if err := s.ChangePassword(t.Context(), p, PasswordInput{OldPassword: "old-password", NewPassword: "new-password-123"}); err != nil || !users.changed {
		t.Fatal("改密参数没有传递到用户业务")
	}
}
