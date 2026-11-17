//go:build integration

package security

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/permission"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/session"
	"github.com/StarsSkyCat/Stars-Admin/internal/testutil"
	token "github.com/StarsSkyCat/Stars-Admin/pkg/jwt"
	"gorm.io/gorm"
)

type failingPolicies struct {
	*permission.Store
	fail atomic.Bool
}

func (p *failingPolicies) Reload(ctx context.Context) error {
	if p.fail.Load() {
		return errors.New("模拟策略重载失败")
	}
	return p.Store.Reload(ctx)
}

type failingRevoker struct {
	*session.Store
	fail atomic.Bool
}

func (s *failingRevoker) RevokeBefore(ctx context.Context, id, version int64) error {
	if s.fail.Load() {
		return errors.New("模拟 Redis 不可用")
	}
	return s.Store.RevokeBefore(ctx, id, version)
}

func TestPostCommitFailureRollbackAndRestartRecovery(t *testing.T) {
	cfg, err := config.Load("../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	id, err := deps.IDs.Next()
	if err != nil {
		t.Fatal(err)
	}
	name := "security" + strconv.FormatInt(id, 10)
	if err := deps.DB.Exec("INSERT INTO sys_user(id,username,nickname,password) VALUES(?,?,?,?)", id, name, "测试", "hash").Error; err != nil {
		t.Fatal(err)
	}
	policies, err := permission.New(deps.DB, cfg.Database.Timeout)
	if err != nil {
		t.Fatal(err)
	}
	fp := &failingPolicies{Store: policies}
	sessions := session.New(deps.Redis, cfg.Redis.KeyPrefix, cfg.Redis.Timeout)
	fr := &failingRevoker{Store: sessions}
	gate := New(deps.DB, fp, fr, deps.Logger, cfg.Database.Timeout)
	project := func(context.Context, *gorm.DB) ([]permission.Rule, error) {
		return []permission.Rule{{Ptype: "g", V0: "user:" + strconv.FormatInt(id, 10), V1: "role:7"}, {Ptype: "p", V0: "role:7", V1: "sys:test:read"}}, nil
	}
	if err := gate.Start(t.Context(), project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := gate.Close(); err != nil {
			t.Error(err)
		}
	})
	tokens := token.New("security-test-secret", "security-test", time.Minute)
	old, err := tokens.Issue(id, name, "", time.Now().Add(time.Hour), false)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessions.Login(t.Context(), id, 1, old, 5, true); err != nil {
		t.Fatal(err)
	}
	// 事务失败必须回滚版本，且不能破坏仍有效的旧策略。
	transactionError := errors.New("模拟数据库事务失败")
	if err := gate.Change(t.Context(), func(tx *gorm.DB) ([]Revocation, error) {
		if err := tx.Exec("UPDATE sys_user SET auth_version=auth_version+1 WHERE id=?", id).Error; err != nil {
			return nil, err
		}
		return nil, transactionError
	}); !errors.Is(err, transactionError) {
		t.Fatal("事务失败未传递")
	}
	var version int64
	if err := deps.DB.Table("sys_user").Where("id = ?", id).Pluck("auth_version", &version).Error; err != nil || version != 1 {
		t.Fatal("数据库事务未回滚")
	}
	if err := gate.Read(func() error { return nil }); err != nil {
		t.Fatal("事务回滚不应破坏旧的有效策略")
	}
	fp.fail.Store(true)
	fr.fail.Store(true)
	if err := gate.Change(t.Context(), func(tx *gorm.DB) ([]Revocation, error) {
		if err := tx.Exec("UPDATE sys_user SET auth_version=2 WHERE id=?", id).Error; err != nil {
			return nil, err
		}
		return []Revocation{{id, 2}}, nil
	}); !errors.Is(err, response.ErrUnavailable) {
		t.Fatal("提交后同步失败必须报告不可用")
	}
	if err := deps.DB.Table("sys_user").Where("id = ?", id).Pluck("auth_version", &version).Error; err != nil || version != 2 {
		t.Fatal("不能把提交后的错误当作事务回滚")
	}
	if gate.Read(func() error { return nil }) == nil || gate.Ready(t.Context()) == nil {
		t.Fatal("同步失败继续放行了鉴权")
	}
	if ok, err := sessions.Validate(t.Context(), id, 2, &old.Access); err != nil || ok {
		t.Fatal("Redis 清理失败时，持久版本仍应拒绝旧 Access")
	}
	// 重试必须保留数据库变更后建立的新版本会话。
	fresh, err := tokens.Issue(id, name, "", time.Now().Add(time.Hour), false)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessions.Login(t.Context(), id, 2, fresh, 5, true); err != nil {
		t.Fatal(err)
	}
	fp.fail.Store(false)
	fr.fail.Store(false)
	deadline := time.Now().Add(4 * time.Second)
	for gate.Ready(t.Context()) != nil && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if gate.Ready(t.Context()) != nil {
		t.Fatal("安全状态未恢复")
	}
	if ok, err := sessions.Validate(t.Context(), id, 2, &fresh.Access); err != nil || !ok {
		t.Fatal("重试误删新版本会话")
	}
	if err := gate.Close(); err != nil {
		t.Fatal(err)
	}
	// 重启必须从业务事实重建失效投影。
	if err := deps.DB.Exec("DELETE FROM casbin_rule").Error; err != nil {
		t.Fatal(err)
	}
	restarted := New(deps.DB, policies, sessions, deps.Logger, cfg.Database.Timeout)
	if err := restarted.Start(t.Context(), project); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := restarted.Close(); err != nil {
			t.Error(err)
		}
	}()
	if ok, err := policies.Allowed(id, "sys:test:read"); err != nil || !ok {
		t.Fatal("重启未恢复当前授权")
	}
	if ok, err := sessions.Rotate(t.Context(), id, 2, &old.Refresh, old.RefreshToken, fresh); err != nil || ok {
		t.Fatal("重启后旧 Refresh 不能恢复")
	}
}

func TestLostCommitAcknowledgementFailsClosed(t *testing.T) {
	cfg, err := config.Load("../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	policies, err := permission.New(deps.DB, cfg.Database.Timeout)
	if err != nil {
		t.Fatal(err)
	}
	sessions := session.New(deps.Redis, cfg.Redis.KeyPrefix, cfg.Redis.Timeout)
	gate := New(deps.DB, policies, sessions, deps.Logger, cfg.Database.Timeout)
	project := func(context.Context, *gorm.DB) ([]permission.Rule, error) { return []permission.Rule{}, nil }
	if err := gate.Start(t.Context(), project); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := gate.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := gate.transact
	gate.transact = func(ctx context.Context, fn func(*gorm.DB) error) error {
		if err := runner(ctx, fn); err != nil {
			return err
		}
		return errors.New("模拟提交成功后确认丢失")
	}
	id, err := deps.IDs.Next()
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.Change(t.Context(), func(tx *gorm.DB) ([]Revocation, error) {
		return nil, tx.Exec("INSERT INTO sys_user(id,username,nickname,password) VALUES(?,?,?,?)", id, "commit"+strconv.FormatInt(id, 10), "测试", "hash").Error
	}); !errors.Is(err, response.ErrUnavailable) {
		t.Fatal("提交确认丢失必须停止授权")
	}
	var count int64
	if err := deps.DB.Table("sys_user").Where("id = ?", id).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("未模拟实际已提交的数据")
	}
	if gate.Ready(t.Context()) == nil || gate.Read(func() error { return nil }) == nil {
		t.Fatal("提交结果不确定时沿用了旧策略")
	}
	gate.transact = runner
	deadline := time.Now().Add(4 * time.Second)
	for gate.Ready(t.Context()) != nil && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if gate.Ready(t.Context()) != nil {
		t.Fatal("提交确认丢失后未恢复")
	}
}
