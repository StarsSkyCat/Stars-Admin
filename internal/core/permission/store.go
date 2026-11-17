// Package permission 提供受控 Casbin 投影存储与线程安全策略加载。
package permission

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/configs"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// Rule 对齐正式迁移中的自定义字段宽度；不执行 AutoMigrate。
type Rule struct {
	ID    uint64 `gorm:"primaryKey;autoIncrement"`
	Ptype string `gorm:"size:100"`
	V0    string `gorm:"size:255"`
	V1    string `gorm:"size:255"`
	V2    string `gorm:"size:255"`
	V3    string `gorm:"size:255"`
	V4    string `gorm:"size:255"`
	V5    string `gorm:"size:255"`
}

// TableName 明确第三方组件表，不使用业务前缀。
func (Rule) TableName() string { return "casbin_rule" }

type boundedAdapter struct {
	*gormadapter.Adapter
	timeout time.Duration
	parent  context.Context
}

func (a boundedAdapter) LoadPolicy(m model.Model) error {
	ctx, cancel := context.WithTimeout(a.parent, a.timeout)
	defer cancel()
	return a.LoadPolicyCtx(ctx, m)
}

// Store 禁止通过 Enforcer AutoSave 独立提交业务授权。
type Store struct {
	mu       sync.RWMutex
	enforcer *casbin.SyncedEnforcer
	adapter  *gormadapter.Adapter
	timeout  time.Duration
}

// New 从现有连接构造 adapter；关闭建表仅影响其独立 GORM 会话。
func New(db *gorm.DB, timeout time.Duration) (*Store, error) {
	clone := db.Session(&gorm.Session{NewDB: true})
	gormadapter.TurnOffAutoMigrate(clone)
	adapter, err := gormadapter.NewAdapterByDBWithCustomTable(clone, &Rule{})
	if err != nil {
		return nil, err
	}
	m, err := model.NewModelFromString(configs.RBACModel)
	if err != nil {
		return nil, err
	}
	e, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		return nil, err
	}
	e.EnableAutoSave(false)
	return &Store{enforcer: e, adapter: adapter, timeout: timeout}, nil
}

// Lock 在读取授权事实前等待先前的策略事务完成，包括提交确认丢失的事务。
func (s *Store) Lock(ctx context.Context, tx *gorm.DB) error {
	// PostgreSQL 事务级表锁属于策略存储例外，不接受客户端标识符。
	return tx.WithContext(ctx).Exec("LOCK TABLE casbin_rule IN EXCLUSIVE MODE").Error
}

// Replace 必须接收业务事务，投影与业务关系一起提交。
func (s *Store) Replace(ctx context.Context, tx *gorm.DB, rules []Rule) error {
	if err := tx.WithContext(ctx).Where("1 = 1").Delete(&Rule{}).Error; err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}
	return tx.WithContext(ctx).CreateInBatches(rules, 200).Error
}

// Reload 在安全协调器的写锁内执行，失败后调用方必须停止鉴权。
func (s *Store) Reload(ctx context.Context) error {
	m, err := model.NewModelFromString(configs.RBACModel)
	if err != nil {
		return err
	}
	candidate, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		return err
	}
	candidate.SetAdapter(boundedAdapter{s.adapter, s.timeout, ctx})
	candidate.EnableAutoSave(false)
	if err := candidate.LoadPolicy(); err != nil {
		return err
	}
	s.mu.Lock()
	s.enforcer = candidate
	s.mu.Unlock()
	return nil
}

// Allowed 使用命名空间主体检查权限，不承担 admin 特例。
func (s *Store) Allowed(id int64, perm string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enforcer.Enforce("user:"+strconv.FormatInt(id, 10), perm)
}

// Membership 为业务服务提供与模型无关的关系快照。
type Membership struct{ UserID, RoleID int64 }

// Grant 为业务服务提供角色与菜单的授权快照。
type Grant struct{ RoleID, MenuID int64 }
