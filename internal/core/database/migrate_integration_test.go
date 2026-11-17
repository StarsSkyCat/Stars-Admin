//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
)

func TestMigrationLifecycleAndConstraints(t *testing.T) {
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
	if status.Version > 0 {
		if err := Migrate(ctx, cfg.Database, "down"); err != nil {
			t.Fatal(err)
		}
	}
	if CheckMigrations(ctx, connection.SQL) == nil {
		t.Fatal("空数据库不能就绪")
	}
	if err := Migrate(ctx, cfg.Database, "up"); err != nil {
		t.Fatal(err)
	}
	if err := CheckMigrations(ctx, connection.SQL); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := connection.SQL.QueryRowContext(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name LIKE 'sys_%'").Scan(&count); err != nil || count != 15 {
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
	}{{1, true}, {-1, true}, {2, false}} {
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
	if _, err := connection.SQL.ExecContext(ctx, "UPDATE public.schema_migrations SET version=1, dirty=false"); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, cfg.Database, "down"); err != nil {
		t.Fatal(err)
	}
	if err := CheckMigrations(ctx, connection.SQL); err == nil {
		t.Fatal("回退后不能就绪")
	}
	if err := Migrate(ctx, cfg.Database, "up"); err != nil {
		t.Fatal(err)
	}
	if err := CheckMigrations(ctx, connection.SQL); err != nil {
		t.Fatal(err)
	}
}
