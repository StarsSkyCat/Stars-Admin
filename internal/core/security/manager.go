// Package security 协调单应用实例中的认证读取与安全状态变更。
package security

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/permission"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Principal 只含可信认证上下文，不接收客户端的角色或操作者字段。
type Principal struct {
	ID                 model.ID   `json:"id"`
	Username           string     `json:"username"`
	RoleIDs            []model.ID `json:"roleIds"`
	AuthVersion        int64      `json:"-"`
	MustChangePassword bool       `json:"mustChangePassword"`
}

// Admin 仅有效的保留角色可以获得管理权限。
func (p Principal) Admin() bool {
	return slices.Contains(p.RoleIDs, 1)
}

// Account 是跨服务认证快照，哈希永不参与 JSON 输出。
type Account struct {
	Principal
	Password string `json:"-"`
	Nickname string `json:"nickname"`
	Status   int    `json:"-"`
}

// Revocation 表示已在数据库事务提交的会话版本下界。
type Revocation struct{ UserID, Version int64 }

// Policies 让投影写入与提交后的加载能够分别验证故障行为。
type Policies interface {
	Lock(context.Context, *gorm.DB) error
	Replace(context.Context, *gorm.DB, []permission.Rule) error
	Reload(context.Context) error
	Allowed(int64, string) (bool, error)
}

// Revoker 只清理数据库已提交的旧版本会话。
type Revoker interface {
	RevokeBefore(context.Context, int64, int64) error
}

// Manager 将业务事务、策略重载与可重试撤销串行协调。
type Manager struct {
	mu       sync.RWMutex
	db       *gorm.DB
	transact func(context.Context, func(*gorm.DB) error) error
	policies Policies
	sessions Revoker
	project  func(context.Context, *gorm.DB) ([]permission.Rule, error)
	pending  map[int64]int64
	rebuild  bool
	reload   bool
	ready    atomic.Bool
	log      *zap.Logger
	timeout  time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// New 构造协调器；Start 必须在模块依赖装配后调用。
func New(db *gorm.DB, policies Policies, sessions Revoker, log *zap.Logger, timeout time.Duration) *Manager {
	return &Manager{db: db, transact: func(ctx context.Context, fn func(*gorm.DB) error) error { return db.WithContext(ctx).Transaction(fn) }, policies: policies, sessions: sessions, log: log, timeout: timeout, pending: map[int64]int64{}}
}

// Start 在接受流量前按事实来源重建投影，并启动受生命周期约束的重试。
func (m *Manager) Start(ctx context.Context, project func(context.Context, *gorm.DB) ([]permission.Rule, error)) error {
	m.project = project
	if err := m.Change(ctx, func(*gorm.DB) ([]Revocation, error) { return nil, nil }); err != nil {
		return err
	}
	worker, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.wg.Go(func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-worker.Done():
				return
			case <-ticker.C:
				if m.ready.Load() {
					continue
				}
				m.mu.Lock()
				retry, c := context.WithTimeout(worker, m.timeout)
				if err := m.synchronize(retry); err != nil {
					m.log.Error("安全状态同步重试失败")
				}
				c()
				m.mu.Unlock()
			}
		}
	})
	return nil
}

// Read 在短暂认证操作期间阻止安全写入，不跨 HTTP handler 持锁。
func (m *Manager) Read(fn func() error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.ready.Load() {
		return response.ErrUnavailable
	}
	return fn()
}

// Change 在同一事务写业务与投影，提交后同步 Redis 和内存策略。
func (m *Manager) Change(ctx context.Context, change func(*gorm.DB) ([]Revocation, error)) error {
	return m.change(ctx, true, change)
}

// Update 串行提交不影响 Casbin 投影的变更，并同步必要的会话撤销。
func (m *Manager) Update(ctx context.Context, change func(*gorm.DB) ([]Revocation, error)) error {
	return m.change(ctx, false, change)
}

func (m *Manager) change(ctx context.Context, project bool, change func(*gorm.DB) ([]Revocation, error)) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	var revoked []Revocation
	prepared := false
	err := m.transact(ctx, func(tx *gorm.DB) error {
		if err := m.policies.Lock(ctx, tx); err != nil {
			return err
		}
		var err error
		revoked, err = change(tx)
		if err != nil {
			return err
		}
		if project {
			rules, err := m.project(ctx, tx)
			if err != nil {
				return err
			}
			if err := m.policies.Replace(ctx, tx, rules); err != nil {
				return err
			}
		}
		prepared = true
		return nil
	})
	if err != nil {
		// 回调完成后的提交错误不能证明未提交，拒绝使用可能陈旧的策略。
		if prepared {
			m.ready.Store(false)
			m.rebuild = true
			for _, r := range revoked {
				m.pending[r.UserID] = max(m.pending[r.UserID], r.Version)
			}
			m.log.Error("数据库提交结果不确定，已停止授权并等待安全状态重同步")
			return response.ErrUnavailable
		}
		return err
	}
	if project {
		m.reload = true
	}
	if len(revoked) == 0 && m.ready.Load() && !m.reload {
		return nil
	}
	m.ready.Store(false)
	for _, r := range revoked {
		m.pending[r.UserID] = max(m.pending[r.UserID], r.Version)
	}
	if err := m.synchronize(ctx); err != nil {
		m.log.Error("数据库已提交，安全状态同步待重试")
		return response.ErrUnavailable
	}
	return nil
}

func (m *Manager) synchronize(ctx context.Context) error {
	if m.rebuild {
		// 等待未知提交事务释放锁后才读事实，避免装载尚未提交的旧投影。
		if err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := m.policies.Lock(ctx, tx); err != nil {
				return err
			}
			rules, err := m.project(ctx, tx)
			if err != nil {
				return err
			}
			return m.policies.Replace(ctx, tx, rules)
		}); err != nil {
			return err
		}
		m.rebuild = false
		m.reload = true
	}
	var combined error
	for id, version := range m.pending {
		if err := m.sessions.RevokeBefore(ctx, id, version); err != nil {
			combined = errors.Join(combined, err)
		} else {
			delete(m.pending, id)
		}
	}
	if m.reload {
		if err := m.policies.Reload(ctx); err != nil {
			combined = errors.Join(combined, err)
		} else {
			m.reload = false
		}
	}
	m.ready.Store(combined == nil)
	return combined
}

// Ready 对外暴露安全依赖的同步状态。
func (m *Manager) Ready(context.Context) error {
	if !m.ready.Load() {
		return response.ErrUnavailable
	}
	return nil
}

// Allow 在完成身份校验后执行权限检查，资源门禁和范围由业务层保证。
func (m *Manager) Allow(p Principal, perm string) error {
	if !m.ready.Load() {
		return response.ErrUnavailable
	}
	if p.Admin() {
		return nil
	}
	ok, err := m.policies.Allowed(int64(p.ID), perm)
	if err != nil {
		return response.ErrUnavailable
	}
	if !ok {
		return response.ErrPermission
	}
	return nil
}

// Close 在 HTTP 排空后停止重试，再由应用关闭数据库和 Redis。
func (m *Manager) Close() error {
	if m.cancel != nil {
		m.cancel()
		m.wg.Wait()
	}
	m.ready.Store(false)
	return nil
}
