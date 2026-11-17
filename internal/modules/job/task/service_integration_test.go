//go:build integration

package task

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/scheduler"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/StarsSkyCat/Stars-Admin/internal/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type memorySink struct{ events chan scheduler.Event }

func (s memorySink) Submit(e scheduler.Event) { s.events <- e }

type failScheduler struct {
	*scheduler.Engine
	fail      atomic.Bool
	attempts  atomic.Int64
	suspended atomic.Int64
}

func (s *failScheduler) Replace(specs []scheduler.Spec) error {
	s.attempts.Add(1)
	if s.fail.Load() {
		return errors.New("模拟计划装载失败")
	}
	return s.Engine.Replace(specs)
}
func (s *failScheduler) Suspend() { s.suspended.Add(1); s.Engine.Suspend() }
func TestCommittedTaskSyncFailureRetriesAndStopsOldPlans(t *testing.T) {
	cfg, err := config.Load("../../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	if err := deps.DB.Exec("TRUNCATE job_task,job_log").Error; err != nil {
		t.Fatal(err)
	}
	registry := scheduler.NewRegistry()
	var count atomic.Int64
	if err := registry.Register("safe", scheduler.Target{Validate: func(string) error { return nil }, Redact: func(string) string { return "{}" }, Run: func(context.Context, string) error { count.Add(1); return nil }}); err != nil {
		t.Fatal(err)
	}
	sink := memorySink{make(chan scheduler.Event, 16)}
	engine := &failScheduler{Engine: scheduler.New(registry, sink, time.Second)}
	core, recorded := observer.New(zap.WarnLevel)
	service := New(t.Context(), deps.DB, deps.IDs, engine, zap.New(core), time.Second)
	defer func() { service.Stop(); _ = engine.Close() }()
	p := security.Principal{ID: 1, RoleIDs: []model.ID{1}}
	row, err := service.Save(t.Context(), p, 0, Input{Name: "同步任务", Cron: "0 0 0 1 1 *", Target: "safe"})
	if err != nil {
		t.Fatal(err)
	}
	engine.fail.Store(true)
	if err := service.Status(t.Context(), p, row.ID, 1); !errors.Is(err, response.ErrUnavailable) {
		t.Fatal("同步失败伪报已生效")
	}
	var stored task
	if err := deps.DB.First(&stored, "id = ?", row.ID).Error; err != nil || stored.Status != 1 {
		t.Fatal("未保留已提交配置")
	}
	if err := service.Ready(t.Context()); !errors.Is(err, response.ErrUnavailable) || engine.suspended.Load() == 0 {
		t.Fatal("故障未停止旧计划或 readiness")
	}
	if _, err := service.Run(t.Context(), p, row.ID); !errors.Is(err, response.ErrUnavailable) {
		t.Fatal("未就绪时仍接受手动执行")
	}
	if recorded.FilterMessage("任务配置已提交，调度同步待重试").Len() != 1 {
		t.Fatal("调度失配缺少告警")
	}
	engine.fail.Store(false)
	deadline := time.Now().Add(3 * time.Second)
	for service.Ready(t.Context()) != nil && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if err := service.Ready(t.Context()); err != nil || engine.attempts.Load() < 3 {
		t.Fatal("调度同步未自动恢复")
	}
	if _, err := service.Run(t.Context(), p, row.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-sink.events:
		if e.Status != 1 || count.Load() != 1 {
			t.Fatal("恢复后执行异常")
		}
	case <-time.After(time.Second):
		t.Fatal("恢复后没有执行")
	}
}
func TestStartupMisfireRunsOncePerProcessAndNotOnEdit(t *testing.T) {
	cfg, err := config.Load("../../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	if err := deps.DB.Exec("TRUNCATE job_task,job_log").Error; err != nil {
		t.Fatal(err)
	}
	if err := deps.DB.Exec("INSERT INTO job_task(id,created_at,updated_at,name,cron,target,params,status,misfire_policy) VALUES(501,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'补跑','0 0 0 1 1 *','safe','{}',1,1),(502,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'跳过','0 0 0 1 1 *','safe','{}',1,2)").Error; err != nil {
		t.Fatal(err)
	}
	registry := scheduler.NewRegistry()
	var count atomic.Int64
	if err := registry.Register("safe", scheduler.Target{Validate: func(string) error { return nil }, Redact: func(string) string { return "{}" }, Run: func(context.Context, string) error { count.Add(1); return nil }}); err != nil {
		t.Fatal(err)
	}
	sink := memorySink{make(chan scheduler.Event, 16)}
	run := func(edit bool) {
		t.Helper()
		engine := scheduler.New(registry, sink, time.Second)
		service := New(t.Context(), deps.DB, deps.IDs, engine, zap.NewNop(), time.Second)
		select {
		case e := <-sink.events:
			if e.TaskID != 501 {
				t.Fatal("启动补跑策略错误")
			}
		case <-time.After(time.Second):
			t.Fatal("缺少启动补跑")
		}
		if edit {
			p := security.Principal{ID: 1, RoleIDs: []model.ID{1}}
			if _, err := service.Save(t.Context(), p, 501, Input{Name: "编辑补跑", Cron: "0 0 0 1 1 *", Target: "safe", MisfirePolicy: 1}); err != nil {
				t.Fatal(err)
			}
		}
		service.Stop()
		if err := engine.Close(); err != nil {
			t.Fatal(err)
		}
	}
	run(true)
	if count.Load() != 1 {
		t.Fatal("编辑触发了额外补跑")
	}
	run(false)
	if count.Load() != 2 {
		t.Fatal("进程重启语义错误")
	}
	if len(sink.events) != 0 {
		t.Fatal("跳过策略产生额外执行")
	}
}
