package scheduler

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/robfig/cron/v3"
)

// Spec 是调度快照，不含数据库模型。
type Spec struct {
	ID         int64
	Name       string
	Group      string
	Cron       string
	Target     string
	Params     string
	Concurrent bool
	Misfire    int
}

// Event 只携带可信注册目标提供的脱敏参数及固定错误分类。
type Event struct {
	TaskID    int64
	Name      string
	Group     string
	Target    string
	Params    string
	Status    int
	Error     string
	CostMS    int
	StartedAt time.Time
}

// Sink 在请求与任务执行链中只能进行非阻塞投递。
type Sink interface{ Submit(Event) }
type running struct {
	count  int
	serial int
}

// Engine 的运行门禁独立于 cron 条目，重载不重置在途状态。
type Engine struct {
	mu       sync.Mutex
	cron     *cron.Cron
	registry *Registry
	sink     Sink
	entries  []cron.EntryID
	running  map[int64]running
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	timeout  time.Duration
	stopping bool
	paused   bool
	revision uint64
}

var parser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// Next 验证严格六位表达式并预览未来五次 UTC 触发，不允许时区覆盖。
func Next(expr string, after time.Time) ([]time.Time, error) {
	if len(expr) > 64 || len(strings.Fields(expr)) != 6 || strings.Contains(expr, "=") {
		return nil, response.ErrInvalidParam
	}
	s, err := parser.Parse(expr)
	if err != nil {
		return nil, response.ErrInvalidParam
	}
	times := make([]time.Time, 0, 5)
	for range 5 {
		after = s.Next(after.UTC())
		if after.IsZero() {
			return nil, response.ErrInvalidParam
		}
		times = append(times, after.UTC())
	}
	return times, nil
}

// New 启动 UTC cron，实际任务由 Replace 装载。
func New(r *Registry, sink Sink, timeout time.Duration) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{cron: cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC)), registry: r, sink: sink, running: map[int64]running{}, ctx: ctx, cancel: cancel, timeout: timeout}
	e.cron.Start()
	return e
}

// Validate 同一校验用于保存、启用、手动与定时入口。
func (e *Engine) Validate(s Spec) error {
	if _, err := Next(s.Cron, time.Now()); err != nil {
		return err
	}
	t, ok := e.registry.get(s.Target)
	if !ok || len(s.Params) > 4096 {
		return response.ErrInvalidParam
	}
	if err := t.Validate(s.Params); err != nil {
		return response.ErrInvalidParam
	}
	return nil
}

// Replace 先完整校验，再替换计划；正在执行的门禁和 context 均保留。
func (e *Engine) Replace(specs []Spec) error {
	for _, s := range specs {
		if err := e.Validate(s); err != nil {
			return err
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopping {
		return response.ErrUnavailable
	}
	for _, id := range e.entries {
		e.cron.Remove(id)
	}
	e.entries = nil
	e.revision++
	revision := e.revision
	for _, s := range specs {
		schedule, err := parser.Parse(s.Cron)
		if err != nil {
			return err
		}
		id := e.cron.Schedule(schedule, cron.FuncJob(func() { _, _ = e.run(s, revision) }))
		e.entries = append(e.entries, id)
	}
	e.paused = false
	return nil
}

// Run 返回 queued/skipped；任务执行不持有 HTTP context，手动和定时共用门禁。
func (e *Engine) Run(s Spec) (string, error) { return e.run(s, 0) }
func (e *Engine) run(s Spec, revision uint64) (string, error) {
	if err := e.Validate(s); err != nil {
		return "", err
	}
	t, _ := e.registry.get(s.Target)
	e.mu.Lock()
	if e.stopping {
		e.mu.Unlock()
		return "", response.ErrUnavailable
	}
	if revision != 0 && revision != e.revision {
		e.mu.Unlock()
		return "skipped", nil
	}
	if e.paused {
		e.mu.Unlock()
		return "", response.ErrUnavailable
	}
	gate := e.running[s.ID]
	if gate.serial > 0 || (!s.Concurrent && gate.count > 0) {
		e.mu.Unlock()
		e.sink.Submit(Event{TaskID: s.ID, Name: s.Name, Group: s.Group, Target: s.Target, Params: redact(t, s.Params), Status: 2, Error: "门禁占用，已跳过", StartedAt: time.Now().UTC()})
		return "skipped", nil
	}
	gate.count++
	if !s.Concurrent {
		gate.serial++
	}
	e.running[s.ID] = gate
	e.wg.Go(func() {
		defer func() {
			e.mu.Lock()
			gate := e.running[s.ID]
			gate.count--
			if !s.Concurrent {
				gate.serial--
			}
			if gate.count == 0 {
				delete(e.running, s.ID)
			} else {
				e.running[s.ID] = gate
			}
			e.mu.Unlock()
		}()
		started := time.Now().UTC()
		ctx, cancel := context.WithTimeout(e.ctx, e.timeout)
		defer cancel()
		event := Event{TaskID: s.ID, Name: s.Name, Group: s.Group, Target: s.Target, Params: redact(t, s.Params), Status: 1, StartedAt: started}
		defer func() {
			if recover() != nil {
				event.Status = 2
				event.Error = "任务发生 panic"
			}
			event.CostMS = int(min(time.Since(started).Milliseconds(), int64(2147483647)))
			e.sink.Submit(event)
		}()
		err := t.Run(ctx, s.Params)
		switch {
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			event.Status = 2
			event.Error = "任务执行超时"
		case errors.Is(ctx.Err(), context.Canceled):
			event.Status = 2
			event.Error = "任务已取消"
		case err != nil:
			event.Status = 2
			event.Error = "任务执行失败"
		}
	})
	e.mu.Unlock()
	return "queued", nil
}

// StopScheduling 关闭入口并取消任务 context，不能提前关闭其数据库依赖。
func (e *Engine) StopScheduling() {
	e.mu.Lock()
	if !e.stopping {
		e.stopping = true
		e.cron.Stop()
		e.cancel()
	}
	e.mu.Unlock()
}

// Close 等待所有可信目标响应取消并退出。
func (e *Engine) Close() error { e.StopScheduling(); e.wg.Wait(); return nil }

// Suspend 移除未来计划但保留在途门禁，提交后同步故障不能继续旧计划。
func (e *Engine) Suspend() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.paused = true
	e.revision++
	for _, id := range e.entries {
		e.cron.Remove(id)
	}
	e.entries = nil
}

func redact(target Target, raw string) (out string) {
	out = "[REDACTED]"
	defer func() {
		if recover() != nil {
			out = "[REDACTED]"
		}
	}()
	text := []rune(target.Redact(raw))
	return string(text[:min(2000, len(text))])
}
