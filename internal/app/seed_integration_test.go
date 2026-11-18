//go:build integration

package app

import (
	"strings"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/database"
	"github.com/StarsSkyCat/Stars-Admin/pkg/hash"
	"go.uber.org/zap"
)

func TestInitialDataMigrationPreservesExistingRecords(t *testing.T) {
	cfg := integrationConfig(t)
	if err := database.Migrate(t.Context(), cfg, "up", firstPassword); err != nil {
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
	migrate := func(direction, password string) {
		t.Helper()
		if err := database.Migrate(t.Context(), cfg, direction, password); err != nil {
			t.Fatal(err)
		}
	}
	migrate("down", "")
	clear()
	migrate("up", firstPassword)
	encoded, err := hash.Password(changedPassword, 4)
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		"DELETE FROM sys_user_role",
		"DELETE FROM sys_role_menu",
		"UPDATE sys_role SET status=2 WHERE id=1",
		"UPDATE sys_menu SET id=100000000000001, path='/system/user' WHERE path='/system/users'",
		"UPDATE sys_menu SET parent_id=100000000000001 WHERE parent_id=101",
		"DELETE FROM sys_menu WHERE perm='sys:user:export'",
		"UPDATE sys_config SET value='false' WHERE key='sys.account.captchaEnabled'",
	} {
		if err := db.ORM.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.ORM.Exec("UPDATE sys_user SET status=2,password=?,pwd_updated_at=? WHERE id=1", encoded, time.Now().UTC()).Error; err != nil {
		t.Fatal(err)
	}
	migrate("down", "")
	migrate("up", "")
	var userMenuCount int64
	if err := db.ORM.Table("sys_menu").Where("path IN ?", []string{"/system/user", "/system/users"}).Count(&userMenuCount).Error; err != nil || userMenuCount != 1 {
		t.Fatal("重复初始化不能创建旧路径对应的重复菜单")
	}
	var parent int64
	if err := db.ORM.Table("sys_menu").Select("parent_id").Where("perm='sys:user:export'").Scan(&parent).Error; err != nil || parent != 100000000000001 {
		t.Fatalf("缺失按钮未使用旧菜单 ID：%d，%v", parent, err)
	}
	var captcha string
	if err := db.ORM.Table("sys_config").Select("value").Where("key='sys.account.captchaEnabled'").Scan(&captcha).Error; err != nil || captcha != "false" {
		t.Fatal("重复迁移覆盖了已有基础配置")
	}
	var account struct {
		Password     string
		Status       int
		PwdUpdatedAt *time.Time
	}
	if err := db.ORM.Table("sys_user").Where("id=1").Scan(&account).Error; err != nil || account.Status != 2 || account.PwdUpdatedAt == nil || !hash.Match(account.Password, changedPassword) {
		t.Fatal("重复迁移覆盖了已有账户状态或密码")
	}
	for _, table := range []string{"sys_user_role", "sys_role_menu"} {
		var count int64
		if err := db.ORM.Table(table).Count(&count).Error; err != nil || count != 0 {
			t.Fatal("重复迁移恢复了已经撤销的授权")
		}
	}
	migrate("down", "")
	clear()
	if err := db.ORM.Exec("INSERT INTO sys_user(id,username,nickname,password) VALUES(1,'occupied','占用','hash')").Error; err != nil {
		t.Fatal(err)
	}
	migrationErr := database.Migrate(t.Context(), cfg, "up", firstPassword)
	if migrationErr == nil || !strings.Contains(migrationErr.Error(), "保留管理员 ID 或账号已被占用") {
		t.Fatalf("保留 ID 被占用时数据迁移应失败：%v", migrationErr)
	}
	if strings.Contains(migrationErr.Error(), "$2a$") {
		t.Fatal("迁移错误泄露了 SQL 中的开发密码哈希")
	}
	var count int64
	if err := db.ORM.Table("sys_role").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("保留标识冲突没有整体回滚")
	}
	clear()
	if err := db.ORM.Exec("UPDATE schema_migrations SET version=5, dirty=false").Error; err != nil {
		t.Fatal(err)
	}
	migrate("up", firstPassword)
}

func TestDevelopmentMigrationInitialPassword(t *testing.T) {
	cfg := integrationConfig(t)
	cfg.Auth.BcryptCost = 4
	cfg.File.LocalDir = t.TempDir()
	if err := database.Migrate(t.Context(), cfg, "up", firstPassword); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(t.Context(), cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.ORM.Exec("TRUNCATE sys_dept,sys_user,sys_role,sys_menu,sys_dict_type,sys_config,sys_login_log,casbin_rule CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(t.Context(), cfg, "down", ""); err != nil {
		t.Fatal(err)
	}
	development := *cfg
	development.App.Env = "dev"
	development.Redis.KeyPrefix += "dev:"
	if err := database.Migrate(t.Context(), &development, "up", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.ORM.Exec("UPDATE sys_config SET value='false' WHERE key='sys.account.captchaEnabled'").Error; err != nil {
		t.Fatal(err)
	}
	const initial = "123456"
	open := func(t *testing.T, developmentMode bool) *App {
		t.Helper()
		selected := cfg
		if developmentMode {
			selected = &development
		}
		a, err := New(t.Context(), selected, zap.NewNop())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := a.Close(); err != nil {
				t.Error(err)
			}
		})
		return a
	}
	t.Run("非开发环境拒绝开发初始密码", func(t *testing.T) {
		a := open(t, false)
		requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "admin", "password": initial}, ""), 401, 2002)
	})
	t.Run("开发首次登录改密后可读取默认菜单", func(t *testing.T) {
		a := open(t, true)
		first := loginUser(t, a, "admin", initial)
		if !first.MustChangePassword {
			t.Fatal("开发初始密码必须先修改")
		}
		requireCode(t, callAPI(t, a, "GET", "/api/sys/menus/tree", nil, first.AccessToken), 403, 2008)
		requireCode(t, callAPI(t, a, "PUT", "/api/sys/auth/password", map[string]string{"oldPassword": initial, "newPassword": changedPassword}, first.AccessToken), 200, 0)
		admin := loginUser(t, a, "admin", changedPassword)
		tree := callAPI(t, a, "GET", "/api/sys/menus/tree", nil, admin.AccessToken)
		requireCode(t, tree, 200, 0)
		for _, entry := range []string{"/system/users/index", "/system/departments/index", "sys:user:grant", "sys:dept:list"} {
			if !strings.Contains(string(tree.Data), entry) {
				t.Fatalf("缺少菜单或权限 %s", entry)
			}
		}
	})
}
