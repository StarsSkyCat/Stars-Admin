// Package log 提供有界任务执行日志、查询与清空能力。
package log

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/scheduler"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type idSource interface{ Next() (int64, error) }

// Service 的投递不等待数据库，停止后可安全排空。
type Service struct {
	repo          repository
	ids           idSource
	queue         chan scheduler.Event
	mu            sync.RWMutex
	closed        bool
	wg            sync.WaitGroup
	dropped       atomic.Int64
	closeDeadline atomic.Int64
	log           *zap.Logger
}

// New 启动单日志 worker。
func New(db *gorm.DB, ids idSource, logger *zap.Logger) *Service {
	s := &Service{repo: repository{db}, ids: ids, queue: make(chan scheduler.Event, 1024), log: logger}
	s.wg.Go(s.worker)
	return s
}

// Submit 满队列只计数和告警，绝不等待持久化。
func (s *Service) Submit(e scheduler.Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return
	}
	select {
	case s.queue <- e:
	default:
		n := s.dropped.Add(1)
		if n == 1 || n%100 == 0 {
			s.log.Warn("任务日志队列已满", zap.Int64("dropped", n))
		}
	}
}
func (s *Service) worker() {
	pending := []entry{}
	flush := func() {
		if len(pending) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if deadline := s.closeDeadline.Load(); deadline > 0 {
			bounded, c := context.WithDeadline(ctx, time.Unix(0, deadline))
			defer c()
			ctx = bounded
		}
		var err error
		for range 3 {
			if err = s.repo.write(ctx, pending); err == nil {
				break
			}
		}
		if err != nil {
			s.dropped.Add(int64(len(pending)))
			s.log.Warn("任务日志重试耗尽", zap.Int64("dropped", s.dropped.Load()))
		}
		pending = pending[:0]
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case e, ok := <-s.queue:
			if !ok {
				flush()
				return
			}
			n, err := s.ids.Next()
			if err != nil {
				s.dropped.Add(1)
				continue
			}
			pending = append(pending, entry{ID: model.ID(n), TaskID: model.ID(e.TaskID), Name: e.Name, TaskGroup: e.Group, Target: e.Target, Params: e.Params, Status: e.Status, ErrorMsg: e.Error, CostMS: e.CostMS, StartedAt: e.StartedAt})
			if len(pending) >= 50 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// List 返回脱敏执行日志。
func (s *Service) List(ctx context.Context, q Query) (response.Page[View], error) {
	if err := q.Normalize(); err != nil {
		return response.Page[View]{}, err
	}
	if q.Status < 0 || q.Status > 2 || (q.OrderBy != "" && q.OrderBy != "id desc") {
		return response.Page[View]{}, response.ErrInvalidParam
	}
	rows, total, err := s.repo.list(ctx, q)
	out := response.Page[View]{List: []View{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, r := range rows {
		out.List = append(out.List, View{ID: r.ID, TaskID: r.TaskID, Name: r.Name, TaskGroup: r.TaskGroup, Target: r.Target, Params: r.Params, Status: r.Status, ErrorMsg: r.ErrorMsg, CostMS: r.CostMS, StartedAt: r.StartedAt})
	}
	return out, err
}

// Clean 物理删除已落库日志，后续执行仍可产生新记录。
func (s *Service) Clean(ctx context.Context) error { return s.repo.clean(ctx) }

// Close 必须在调度器等待全部在途任务之后调用。
func (s *Service) Close() error {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		s.closeDeadline.Store(time.Now().Add(3 * time.Second).UnixNano())
		close(s.queue)
	}
	s.mu.Unlock()
	s.wg.Wait()
	return nil
}

// Dropped 返回进程累计丢弃计数。
func (s *Service) Dropped() int64 { return s.dropped.Load() }
