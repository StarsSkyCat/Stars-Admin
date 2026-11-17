package scheduler

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
)

type eventSink struct{ events chan Event }

func (s eventSink) Submit(e Event) { s.events <- e }
func receive(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(3 * time.Second):
		t.Fatal("等待任务日志超时")
		return Event{}
	}
}
func TestCronUsesStrictSecondsUTCAndRejectsNoFuture(t *testing.T) {
	after := time.Date(2026, 11, 17, 20, 0, 0, 0, time.FixedZone("east", 8*3600))
	next, err := Next("0 */5 * * * *", after)
	if err != nil || len(next) != 5 || next[0].Format(time.RFC3339) != "2026-11-17T12:05:00Z" {
		t.Fatalf("UTC预览无效 %v %v", next, err)
	}
	for _, expr := range []string{"* * * * *", "CRON_TZ=Asia/Shanghai 0 * * * * *", "@every 1s", "0 0 0 31 2 *", "bad"} {
		if _, err := Next(expr, after); !errors.Is(err, response.ErrInvalidParam) {
			t.Fatalf("非法或无未来的Cron被接受 %s", expr)
		}
	}
}
func TestTimerAndManualShareGateAcrossReload(t *testing.T) {
	registry := NewRegistry()
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	var count atomic.Int64
	if err := registry.Register("controlled", Target{Validate: func(string) error { return nil }, Redact: func(string) string { return "[REDACTED]" }, Run: func(ctx context.Context, _ string) error {
		count.Add(1)
		started <- struct{}{}
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}); err != nil {
		t.Fatal(err)
	}
	sink := eventSink{make(chan Event, 32)}
	engine := New(registry, sink, 10*time.Second)
	t.Cleanup(func() { _ = engine.Close() })
	spec := Spec{ID: 1, Name: "门禁", Cron: "* * * * * *", Target: "controlled", Params: "{}"}
	if err := engine.Replace([]Spec{spec}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("真实定时入口没有执行")
	}
	if state, err := engine.Run(spec); err != nil || state != "skipped" {
		t.Fatal("手动执行绕过定时门禁")
	}
	modified := spec
	modified.Concurrent = true
	if err := engine.Replace([]Spec{modified}); err != nil {
		t.Fatal(err)
	}
	if state, err := engine.Run(modified); err != nil || state != "skipped" {
		t.Fatal("Reload重置了正在运行的非并发门禁")
	}
	if count.Load() != 1 {
		t.Fatal("门禁未共享")
	}
	engine.mu.Lock()
	oldRevision := engine.revision
	engine.mu.Unlock()
	engine.Suspend()
	if state, err := engine.run(spec, oldRevision); err != nil || state != "skipped" {
		t.Fatal("旧计划触发未丢弃")
	}
	close(release)
	engine.StopScheduling()
	if err := engine.Close(); err != nil {
		t.Fatal(err)
	}
	if count.Load() != 1 {
		t.Fatal("关闭后又执行任务")
	}
	if state, err := engine.Run(spec); !errors.Is(err, response.ErrUnavailable) || state != "" {
		t.Fatal("停机后仍接受任务")
	}
	event := receive(t, sink.events)
	if event.Status != 2 || !strings.Contains(event.Error, "跳过") {
		t.Fatal("门禁跳过伪报成功")
	}
}
func TestPanicTimeoutAndCancellationAreRecordedSafely(t *testing.T) {
	for _, mode := range []string{"panic", "timeout", "cancel", "error"} {
		t.Run(mode, func(t *testing.T) {
			registry := NewRegistry()
			started := make(chan struct{})
			err := registry.Register("target", Target{Validate: func(string) error { return nil }, Redact: func(string) string { return "[REDACTED]" }, Run: func(ctx context.Context, raw string) error {
				close(started)
				switch mode {
				case "panic":
					panic(raw)
				case "error":
					return errors.New(raw)
				default:
					<-ctx.Done()
					return ctx.Err()
				}
			}})
			if err != nil {
				t.Fatal(err)
			}
			sink := eventSink{make(chan Event, 4)}
			timeout := 20 * time.Millisecond
			if mode == "cancel" {
				timeout = time.Second
			}
			engine := New(registry, sink, timeout)
			spec := Spec{ID: 2, Cron: "0 0 0 1 1 *", Target: "target", Params: "secret-value"}
			if state, err := engine.Run(spec); err != nil || state != "queued" {
				t.Fatal(err)
			}
			<-started
			if mode == "cancel" {
				engine.StopScheduling()
			}
			event := receive(t, sink.events)
			if event.Status != 2 || event.Error == "" || strings.Contains(event.Params+event.Error, "secret-value") {
				t.Fatal("任务失败未正确脱敏记录")
			}
			if mode == "timeout" && !strings.Contains(event.Error, "超时") {
				t.Fatal("超时被伪报成功")
			}
			if err := engine.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestRegistryRejectsUnknownAndDuplicateTargets(t *testing.T) {
	r := NewRegistry()
	target := Target{Validate: func(string) error { return nil }, Redact: func(string) string { return "{}" }, Run: func(context.Context, string) error { return nil }}
	if err := r.Register("demo", target); err != nil {
		t.Fatal(err)
	}
	if err := r.Register("demo", target); err == nil {
		t.Fatal("重复注册被接受")
	}
	sink := eventSink{make(chan Event, 4)}
	engine := New(r, sink, time.Second)
	defer func() { _ = engine.Close() }()
	if err := engine.Validate(Spec{Cron: "* * * * * *", Target: "sh"}); !errors.Is(err, response.ErrInvalidParam) {
		t.Fatal("未知目标被接受")
	}
}
