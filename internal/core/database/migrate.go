package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/migrations"
	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// MigrationStatus 只用于 CLI 和内部就绪判定。
type MigrationStatus struct {
	Version  int64
	Dirty    bool
	Expected uint
}

// LatestVersion 从二进制内嵌迁移计算期望版本。
func LatestVersion() (uint, error) {
	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		return 0, fmt.Errorf("读取内嵌迁移失败: %w", err)
	}
	var latest uint
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		prefix, _, _ := strings.Cut(entry.Name(), "_")
		version, err := strconv.ParseUint(prefix, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("迁移编号无效: %w", err)
		}
		latest = max(latest, uint(version))
	}
	if latest == 0 {
		return 0, errors.New("没有可执行的正式迁移")
	}
	return latest, nil
}

// ReadStatus 查询版本，不创建迁移表、不改变数据库结构。
func ReadStatus(ctx context.Context, db *sql.DB) (MigrationStatus, error) {
	expected, err := LatestVersion()
	if err != nil {
		return MigrationStatus{}, err
	}
	status := MigrationStatus{Expected: expected}
	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.schema_migrations') IS NOT NULL").Scan(&exists); err != nil {
		return status, fmt.Errorf("读取迁移状态失败: %w", err)
	}
	if !exists {
		return status, nil
	}
	err = db.QueryRowContext(ctx, "SELECT version, dirty FROM public.schema_migrations LIMIT 1").Scan(&status.Version, &status.Dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return status, nil
	}
	if err != nil {
		return status, fmt.Errorf("读取迁移版本失败: %w", err)
	}
	return status, nil
}

// CheckMigrations 禁止未升级、脏版本或超前结构启动 HTTP 服务。
func CheckMigrations(ctx context.Context, db *sql.DB) error {
	status, err := ReadStatus(ctx, db)
	if err != nil {
		return err
	}
	if status.Dirty || status.Version != int64(status.Expected) {
		return fmt.Errorf("迁移未就绪：当前版本 %d，期望版本 %d，dirty=%t；请先核实并执行迁移", status.Version, status.Expected, status.Dirty)
	}
	return nil
}

// Migrate 使用独立连接执行显式的升级或单版回退，不依赖 Redis。
func Migrate(ctx context.Context, c config.Database, direction string) (err error) {
	if direction != "up" && direction != "down" {
		return errors.New("迁移动作只能是 up 或 down")
	}
	connection, err := Open(ctx, c)
	if err != nil {
		return err
	}
	driver, err := migratepg.WithInstance(connection.SQL, &migratepg.Config{DatabaseName: c.DBName, SchemaName: "public", StatementTimeout: c.Timeout})
	if err != nil {
		_ = connection.Close()
		return fmt.Errorf("初始化迁移驱动失败: %w", err)
	}
	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		_ = driver.Close()
		return fmt.Errorf("初始化迁移源失败: %w", err)
	}
	manager, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		_ = source.Close()
		_ = driver.Close()
		return fmt.Errorf("创建迁移管理器失败: %w", err)
	}
	manager.LockTimeout = c.Timeout
	defer func() {
		sourceErr, databaseErr := manager.Close()
		err = errors.Join(err, sourceErr, databaseErr)
	}()
	stop := context.AfterFunc(ctx, func() {
		select {
		case manager.GracefulStop <- true:
		default:
		}
	})
	defer stop()
	if direction == "up" {
		err = manager.Up()
	} else {
		err = manager.Steps(-1)
	}
	if errors.Is(err, migrate.ErrNoChange) || errors.Is(err, migrate.ErrNilVersion) {
		return nil
	}
	return err
}
