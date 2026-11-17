package audit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/middleware"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"go.uber.org/zap"
)

type operation struct {
	model.LogBase
	UserID   model.ID
	Username string
	Module   string
	Action   string
	TraceID  string
	Method   string
	URL      string
	IP       string
	Response string
	Request  string
	Status   int
	CostMS   int
	OperAt   time.Time
}

func (operation) TableName() string { return "sys_oper_log" }

type worker struct {
	queue   chan middleware.AuditEvent
	wg      sync.WaitGroup
	mu      sync.RWMutex
	closed  bool
	dropped atomic.Int64
	log     *zap.Logger
}

// Start 启动单 worker，按 50 条或一秒批量写入。
func (s *Service) Start(log *zap.Logger) {
	s.worker = &worker{queue: make(chan middleware.AuditEvent, 1024), log: log}
	s.worker.wg.Go(func() {
		pending := []operation{}
		flush := func() {
			if len(pending) == 0 {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var err error
			for range 3 {
				if err = s.repo.operations(ctx, pending); err == nil {
					break
				}
			}
			if err != nil {
				s.worker.dropped.Add(int64(len(pending)))
				s.worker.log.Warn("操作日志写入重试耗尽", zap.Int64("dropped", s.worker.dropped.Load()))
			}
			pending = pending[:0]
		}
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case event, ok := <-s.worker.queue:
				if !ok {
					flush()
					return
				}
				id, err := s.ids.Next()
				if err != nil {
					s.worker.dropped.Add(1)
					continue
				}
				pending = append(pending, operation{ID: model.ID(id), UserID: event.UserID, Username: event.Username, Module: event.Module, Action: event.Action, TraceID: event.TraceID, Method: event.Method, URL: event.URL, IP: event.IP, Response: event.Response, Status: event.Status, CostMS: event.CostMS, OperAt: event.OperAt})
				if len(pending) >= 50 {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	})
}

// Submit 满队列立即丢弃并计数，不等待数据库。
func (s *Service) Submit(event middleware.AuditEvent) {
	w := s.worker
	if w == nil {
		return
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.closed {
		return
	}
	select {
	case w.queue <- event:
	default:
		total := w.dropped.Add(1)
		if total == 1 || total%100 == 0 {
			w.log.Warn("操作日志队列已满", zap.Int64("dropped", total))
		}
	}
}

// Close 在请求生产者停止后排空队列，再允许数据库关闭。
func (s *Service) Close() error {
	if w := s.worker; w != nil {
		w.mu.Lock()
		if !w.closed {
			w.closed = true
			close(w.queue)
		}
		w.mu.Unlock()
		w.wg.Wait()
	}
	return nil
}
