// Package scheduler 提供单进程 UTC 调度、受控目标注册及统一执行门禁。
package scheduler

import (
	"context"
	"errors"
	"regexp"
	"sync"
)

// Target 的校验与审计摘要都由可信注册代码提供，参数原文不会自动进入日志。
type Target struct {
	Validate func(string) error
	Redact   func(string) string
	Run      func(context.Context, string) error
}

// Registry 在启动阶段注册目标，运行阶段只读。
type Registry struct {
	mu      sync.RWMutex
	targets map[string]Target
}

// NewRegistry 创建显式白名单。
func NewRegistry() *Registry { return &Registry{targets: map[string]Target{}} }

var targetName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.-]{0,254}$`)

// Register 拒绝重复和不完整注册，不支持 shell 或动态代码。
func (r *Registry) Register(name string, t Target) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !targetName.MatchString(name) || t.Validate == nil || t.Redact == nil || t.Run == nil {
		return errors.New("任务注册无效")
	}
	if _, ok := r.targets[name]; ok {
		return errors.New("任务目标重复")
	}
	r.targets[name] = t
	return nil
}
func (r *Registry) get(name string) (Target, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.targets[name]
	return t, ok
}
