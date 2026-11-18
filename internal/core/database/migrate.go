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
	"github.com/StarsSkyCat/Stars-Admin/pkg/hash"
	"github.com/golang-migrate/migrate/v4"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"golang.org/x/crypto/bcrypt"
)

const schemaBaselineVersion = 5
const initialDataVersion = 6

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
func ReadStatus(ctx context.Context, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (MigrationStatus, error) {
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

// Migrate 使用独立连接执行升级或单版回退。initialPassword 只用于首次数据迁移。
func Migrate(ctx context.Context, cfg *config.Config, direction, initialPassword string) (err error) {
	if direction != "up" && direction != "down" {
		return errors.New("迁移动作只能是 up 或 down")
	}
	c := cfg.Database
	connection, err := Open(ctx, c)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, connection.Close()) }()
	setupCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	conn, err := connection.SQL.Conn(setupCtx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	status, err := ReadStatus(setupCtx, conn)
	if err != nil {
		return err
	}
	if status.Version > 0 && status.Version < schemaBaselineVersion {
		return fmt.Errorf("旧迁移版本 %d，dirty=%t；请用包含原迁移的版本升级至 %d", status.Version, status.Dirty, schemaBaselineVersion)
	}
	if direction == "up" && !status.Dirty && status.Version < initialDataVersion {
		if err := setInitialPassword(setupCtx, conn, status.Version, cfg.App.Env == "dev", initialPassword); err != nil {
			return err
		}
	}
	driver, err := migratepg.WithConnection(setupCtx, conn, &migratepg.Config{DatabaseName: c.DBName, SchemaName: "public", StatementTimeout: c.Timeout})
	if err != nil {
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
	if failure, ok := errors.AsType[migratedb.Error](err); ok {
		// 驱动错误包含整段 SQL，输出底层错误以免记录种子哈希。
		return fmt.Errorf("执行 PostgreSQL 迁移失败: %w", failure.OrigErr)
	}
	return err
}

func setInitialPassword(ctx context.Context, conn *sql.Conn, version int64, development bool, password string) error {
	if version == schemaBaselineVersion {
		var exists bool
		if err := conn.QueryRowContext(ctx, "SELECT EXISTS (SELECT FROM public.sys_user WHERE id = 1 AND username = 'admin')").Scan(&exists); err != nil {
			return err
		}
		if exists {
			return nil
		}
	}
	var encoded string
	if password != "" {
		var err error
		encoded, err = hash.Password(password, bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("STARS_SEED_ADMIN_PASSWORD 无效: %w", err)
		}
	} else if !development {
		return errors.New("首次数据迁移必须提供 STARS_SEED_ADMIN_PASSWORD")
	}
	// 哈希与环境标记仅对本次迁移的专用 PostgreSQL 连接有效。
	_, err := conn.ExecContext(ctx, `SELECT set_config('stars_admin.initial_password_hash', $1, false),
		set_config('stars_admin.development', CASE WHEN $2::boolean THEN 'on' ELSE 'off' END, false)`, encoded, development)
	return err
}
