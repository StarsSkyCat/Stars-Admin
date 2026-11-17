//go:build integration

package app

import (
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/database"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys"
	"github.com/StarsSkyCat/Stars-Admin/pkg/hash"
	"github.com/StarsSkyCat/Stars-Admin/pkg/snowflake"
	"go.uber.org/zap"
)

func TestM2SeedRollbackIdempotenceAndReservedIdentity(t *testing.T) {
	cfg := integrationConfig(t)
	cfg.Auth.BcryptCost = 4
	if err := database.Migrate(t.Context(), cfg.Database, "up"); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(t.Context(), cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}()
	clear := func() {
		t.Helper()
		if err := db.ORM.Exec("TRUNCATE sys_dept,sys_user,sys_role,sys_menu,sys_dict_type,sys_config,sys_login_log,casbin_rule RESTART IDENTITY CASCADE").Error; err != nil {
			t.Fatal(err)
		}
	}
	clear()
	ids, err := snowflake.New(0)
	if err != nil {
		t.Fatal(err)
	}
	deps := core.Deps{Config: cfg, DB: db.ORM, IDs: ids, Logger: zap.NewNop()}
	for _, password := range []string{"", "too-short"} {
		if err := sys.Seed(t.Context(), deps, password); err == nil {
			t.Fatal("首次 seed 不能接受缺失或弱密码")
		}
		var count int64
		if err := db.ORM.Table("sys_dept").Count(&count).Error; err != nil || count != 0 {
			t.Fatal("seed 失败没有整体回滚")
		}
	}
	if err := sys.Seed(t.Context(), deps, firstPassword); err != nil {
		t.Fatal(err)
	}
	encoded, err := hash.Password(changedPassword, 4)
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"DELETE FROM sys_user_role", "DELETE FROM sys_role_menu", "UPDATE sys_role SET status=2 WHERE id=1"} {
		if err := db.ORM.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.ORM.Exec("UPDATE sys_user SET status=2,password=?,pwd_updated_at=? WHERE id=1", encoded, time.Now().UTC()).Error; err != nil {
		t.Fatal(err)
	}
	if err := sys.Seed(t.Context(), deps, "Ignored-Password-789"); err != nil {
		t.Fatal(err)
	}
	var account struct {
		Password     string
		Status       int
		PwdUpdatedAt *time.Time
	}
	if err := db.ORM.Table("sys_user").Where("id=1").Scan(&account).Error; err != nil || account.Status != 2 || account.PwdUpdatedAt == nil || !hash.Match(account.Password, changedPassword) {
		t.Fatal("重复 seed 覆盖了已有账户状态或密码")
	}
	for _, table := range []string{"sys_user_role", "sys_role_menu"} {
		var count int64
		if err := db.ORM.Table(table).Count(&count).Error; err != nil || count != 0 {
			t.Fatal("重复 seed 恢复了已经撤销的授权")
		}
	}
	clear()
	if err := db.ORM.Exec("INSERT INTO sys_user(id,username,nickname,password) VALUES(1,'occupied','占用','hash')").Error; err != nil {
		t.Fatal(err)
	}
	if err := sys.Seed(t.Context(), deps, firstPassword); err == nil {
		t.Fatal("保留 ID 被占用时 seed 未拒绝")
	}
	var count int64
	if err := db.ORM.Table("sys_role").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("保留标识冲突没有整体回滚")
	}
}
