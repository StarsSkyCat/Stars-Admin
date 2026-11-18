//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/pkg/hash"
)

func TestMigrationLifecycleAndConstraints(t *testing.T) {
	const initialPassword = "Test-Initial-Password-123"
	cfg, err := config.Load("../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Env != "test" || cfg.Database.DBName != "stars_admin_test" {
		t.Fatal("禁止在非隔离测试库执行迁移测试")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	// 测试项目可能已被其他非破坏性集成测试升级，先回退到空结构。
	connection, err := Open(ctx, cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Error(err)
		}
	})
	status, err := ReadStatus(ctx, connection.SQL)
	if err != nil {
		t.Fatal(err)
	}
	for status.Version > 0 {
		if err := Migrate(ctx, cfg, "down", ""); err != nil {
			t.Fatal(err)
		}
		status, err = ReadStatus(ctx, connection.SQL)
		if err != nil {
			t.Fatal(err)
		}
	}
	if CheckMigrations(ctx, connection.SQL) == nil {
		t.Fatal("空数据库不能就绪")
	}
	for _, password := range []string{"", "too-short"} {
		if err := Migrate(ctx, cfg, "up", password); err == nil {
			t.Fatal("非开发环境首次迁移不能接受缺失或弱密码")
		}
		status, err := ReadStatus(ctx, connection.SQL)
		if err != nil || status.Version != 0 || status.Dirty {
			t.Fatalf("密码校验失败时应保留空库：%+v，%v", status, err)
		}
	}
	if err := Migrate(ctx, cfg, "up", initialPassword); err != nil {
		t.Fatal(err)
	}
	if err := CheckMigrations(ctx, connection.SQL); err != nil {
		t.Fatal(err)
	}
	// 从数据版回退只回退版本记录，回退结构基线会清空业务表。
	for _, expected := range []int64{schemaBaselineVersion, 0} {
		if err := Migrate(ctx, cfg, "down", ""); err != nil {
			t.Fatal(err)
		}
		status, err := ReadStatus(ctx, connection.SQL)
		if err != nil || status.Version != expected || status.Dirty {
			t.Fatalf("逐版回退状态错误：%+v，%v", status, err)
		}
	}
	if err := Migrate(ctx, cfg, "up", initialPassword); err != nil {
		t.Fatal(err)
	}
	var count int
	var encoded string
	if err := connection.SQL.QueryRowContext(ctx, "SELECT password FROM sys_user WHERE id=1 AND pwd_updated_at IS NULL").Scan(&encoded); err != nil || !hash.Match(encoded, initialPassword) {
		t.Fatalf("未创建需要首登改密的管理员：%v", err)
	}
	if err := connection.SQL.QueryRowContext(ctx, "SELECT count(*) FROM sys_role_menu WHERE role_id=1").Scan(&count); err != nil || count != 81 {
		t.Fatalf("管理员菜单授权数量错误：%d，%v", count, err)
	}
	if err := connection.SQL.QueryRowContext(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name LIKE 'sys_%'").Scan(&count); err != nil || count != 16 {
		t.Fatalf("系统表数量错误：%d，%v", count, err)
	}
	var zone string
	if err := connection.SQL.QueryRowContext(ctx, "SHOW TimeZone").Scan(&zone); err != nil || zone != "UTC" {
		t.Fatalf("连接时区错误：%s，%v", zone, err)
	}
	for _, query := range []string{
		"INSERT INTO sys_dept(id,name) VALUES(10,'根部门')",
		"INSERT INTO sys_user(id,username,nickname,password,dept_id) VALUES(11,'alice','测试用户','hash',10)",
		"INSERT INTO sys_role(id,name,code) VALUES(12,'测试角色','worker')",
		"INSERT INTO sys_user_role(id,user_id,role_id) VALUES(13,11,12)",
	} {
		if _, err := connection.SQL.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{
		"INSERT INTO sys_user(id,username,nickname,password) VALUES(14,'alice','重复用户','hash')",
		"INSERT INTO sys_user(id,username,nickname,password,dept_id) VALUES(14,'other','无效部门','hash',999)",
		"INSERT INTO sys_role(id,name,code,status) VALUES(14,'无效状态','invalid',3)",
		"INSERT INTO sys_role(id,name,code,data_scope) VALUES(14,'无效范围','invalid',0)",
		"INSERT INTO sys_user_role(id,user_id,role_id) VALUES(14,11,12)",
		"DELETE FROM sys_user WHERE id=11",
	} {
		if _, err := connection.SQL.ExecContext(ctx, query); err == nil {
			t.Fatal("数据库约束或删除保护未生效")
		}
	}
	for _, query := range []string{
		"DELETE FROM sys_user_role WHERE id=13",
		"INSERT INTO sys_user_role(id,user_id,role_id) VALUES(15,11,12)",
		"UPDATE sys_user SET deleted_at=CURRENT_TIMESTAMP WHERE id=11",
	} {
		if _, err := connection.SQL.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := connection.SQL.ExecContext(ctx, "INSERT INTO sys_user(id,username,nickname,password) VALUES(16,'alice','软删后重复','hash')"); err == nil {
		t.Fatal("软删除后业务键仍应保留")
	}
	for _, state := range []struct {
		version int
		dirty   bool
	}{{1, true}, {-1, true}, {initialDataVersion + 1, false}} {
		if _, err := connection.SQL.ExecContext(ctx, "UPDATE public.schema_migrations SET version=$1, dirty=$2", state.version, state.dirty); err != nil {
			t.Fatal(err)
		}
		status, err := ReadStatus(ctx, connection.SQL)
		if err != nil || status.Version != int64(state.version) || status.Dirty != state.dirty {
			t.Fatal("必须能读取回退失败时的负迁移版本与 dirty 状态")
		}
		if CheckMigrations(ctx, connection.SQL) == nil {
			t.Fatal("脏版本或超前结构未拒绝")
		}
	}
	if _, err := connection.SQL.ExecContext(ctx, "UPDATE public.schema_migrations SET version=4, dirty=false"); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, cfg, "up", initialPassword); err == nil {
		t.Fatal("低于结构基线的旧库应由旧版本继续升级")
	}
	status, err = ReadStatus(ctx, connection.SQL)
	if err != nil || status.Version != 4 || status.Dirty {
		t.Fatalf("旧库版本记录被修改：%+v，%v", status, err)
	}
	if _, err := connection.SQL.ExecContext(ctx, "UPDATE public.schema_migrations SET version=$1, dirty=false", initialDataVersion); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, cfg, "down", ""); err != nil {
		t.Fatal(err)
	}
	if err := CheckMigrations(ctx, connection.SQL); err == nil {
		t.Fatal("回退后不能就绪")
	}
	if err := Migrate(ctx, cfg, "up", ""); err != nil {
		t.Fatal(err)
	}
	if err := CheckMigrations(ctx, connection.SQL); err != nil {
		t.Fatal(err)
	}
	if err := connection.SQL.QueryRowContext(ctx, "SELECT count(*) FROM sys_user WHERE id IN (1, 11)").Scan(&count); err != nil || count != 2 {
		t.Fatalf("基础数据回退和重跑应保留已有账号：%d，%v", count, err)
	}
}
