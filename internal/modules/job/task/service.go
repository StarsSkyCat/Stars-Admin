package task

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/scheduler"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type idSource interface{ Next() (int64, error) }

// Scheduler 是可替换的提交后调度同步能力。
type Scheduler interface {
	Suspend()
	Validate(scheduler.Spec) error
	Replace([]scheduler.Spec) error
	Run(scheduler.Spec) (string, error)
}

// Service 串行提交配置和同步计划，同步故障可从就绪探针观察并重试。
type Service struct {
	repo     repository
	ids      idSource
	engine   Scheduler
	mu       sync.Mutex
	ready    atomic.Bool
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	log      *zap.Logger
	timeout  time.Duration
	stopping atomic.Bool
}

// New 加载启用任务；错误配置保留管理修复入口，但不报告就绪。
func New(ctx context.Context, db *gorm.DB, ids idSource, engine Scheduler, log *zap.Logger, timeout time.Duration) *Service {
	s := &Service{repo: repository{db}, ids: ids, engine: engine, log: log, timeout: timeout}
	rows, err := s.repo.enabled(ctx)
	if err == nil {
		err = s.reload(ctx)
	}
	if err != nil {
		log.Error("任务初次装载失败，调度未就绪")
	} else {
		for _, r := range rows {
			if r.MisfirePolicy == 1 {
				if _, err := engine.Run(spec(r)); err != nil {
					log.Warn("启动补跑投递失败")
				}
			}
		}
	}
	worker, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Go(func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-worker.Done():
				return
			case <-ticker.C:
				if s.ready.Load() || s.stopping.Load() {
					continue
				}
				s.mu.Lock()
				ctx, c := context.WithTimeout(worker, s.timeout)
				if err := s.reload(ctx); err != nil {
					log.Warn("调度同步重试失败")
				}
				c()
				s.mu.Unlock()
			}
		}
	})
	return s
}
func (s *Service) reload(ctx context.Context) error {
	s.engine.Suspend()
	defer func() {
		if !s.ready.Load() {
			s.engine.Suspend()
		}
	}()
	s.ready.Store(false)
	rows, err := s.repo.enabled(ctx)
	if err != nil {
		return err
	}
	specs := make([]scheduler.Spec, 0, len(rows))
	for _, r := range rows {
		specs = append(specs, spec(r))
	}
	if err := s.engine.Replace(specs); err != nil {
		return err
	}
	s.ready.Store(true)
	return nil
}

// Ready 不向探针返回任务参数或内部错误。
func (s *Service) Ready(context.Context) error {
	if !s.ready.Load() || s.stopping.Load() {
		return response.ErrUnavailable
	}
	return nil
}
func (s *Service) change(ctx context.Context, fn func(*gorm.DB) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping.Load() {
		return response.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	s.ready.Store(false)
	s.engine.Suspend()
	prepared := false
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := fn(tx); err != nil {
			return err
		}
		prepared = true
		return nil
	})
	if err != nil {
		if prepared {
			s.ready.Store(false)
			s.engine.Suspend()
			s.log.Error("任务配置提交结果不确定，等待重同步")
			return response.ErrUnavailable
		}
		if syncErr := s.reload(ctx); syncErr != nil {
			s.log.Error("任务事务失败后恢复调度失败")
			return response.ErrUnavailable
		}
		return err
	}
	if err := s.reload(ctx); err != nil {
		s.log.Error("任务配置已提交，调度同步待重试")
		return response.ErrUnavailable
	}
	return nil
}

// Save 新任务默认暂停；编辑启用任务保留状态并重建计划。
func (s *Service) Save(ctx context.Context, p security.Principal, id model.ID, in Input) (View, error) {
	if !p.Admin() {
		return View{}, response.ErrPermission
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len([]rune(in.Name)) > 64 {
		return View{}, response.ErrInvalidParam
	}
	if in.TaskGroup == "" {
		in.TaskGroup = "default"
	}
	if in.Params == "" {
		in.Params = "{}"
	}
	if in.MisfirePolicy == 0 {
		in.MisfirePolicy = 2
	}
	if len(in.TaskGroup) > 64 || (in.MisfirePolicy != 1 && in.MisfirePolicy != 2) {
		return View{}, response.ErrInvalidParam
	}
	var row task
	err := s.change(ctx, func(tx *gorm.DB) error {
		if id == 0 {
			n, err := s.ids.Next()
			if err != nil {
				return err
			}
			row = task{ID: model.ID(n), CreatedBy: p.ID, Status: 2}
		} else {
			var err error
			row, err = s.repo.get(ctx, tx, id)
			if err != nil {
				return httpx.DBError(err)
			}
		}
		row.Name = in.Name
		row.TaskGroup = in.TaskGroup
		row.Cron = in.Cron
		row.Target = in.Target
		row.Params = in.Params
		row.MisfirePolicy = in.MisfirePolicy
		row.Concurrent = in.Concurrent
		row.Remark = in.Remark
		row.UpdatedBy = p.ID
		if err := s.engine.Validate(spec(row)); err != nil {
			return err
		}
		return httpx.DBError(s.repo.save(ctx, tx, &row, id == 0))
	})
	return view(row), err
}

// List 分页查询任务，排序列固定映射。
func (s *Service) List(ctx context.Context, q Query) (response.Page[View], error) {
	if err := q.Normalize(); err != nil {
		return response.Page[View]{}, err
	}
	column, desc, err := q.Sort(map[string]string{"id": "id", "name": "name", "createdAt": "created_at"})
	if err != nil || q.Status < 0 || q.Status > 2 {
		return response.Page[View]{}, response.ErrInvalidParam
	}
	if column == "" {
		column = "id"
	}
	rows, total, err := s.repo.list(ctx, q, column, desc)
	out := response.Page[View]{List: []View{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, r := range rows {
		out.List = append(out.List, view(r))
	}
	return out, err
}

// Get 返回配置详情，管理入口默认仅 admin。
func (s *Service) Get(ctx context.Context, id model.ID) (View, error) {
	r, e := s.repo.get(ctx, nil, id)
	return view(r), httpx.DBError(e)
}

// Status 启用前重新校验当前配置，修改不触发启动补跑。
func (s *Service) Status(ctx context.Context, p security.Principal, id model.ID, status int) error {
	if !p.Admin() {
		return response.ErrPermission
	}
	if status != 1 && status != 2 {
		return response.ErrInvalidParam
	}
	return s.change(ctx, func(tx *gorm.DB) error {
		r, err := s.repo.get(ctx, tx, id)
		if err != nil {
			return httpx.DBError(err)
		}
		if status == 1 {
			if err := s.engine.Validate(spec(r)); err != nil {
				return err
			}
		}
		r.Status = status
		r.UpdatedBy = p.ID
		return s.repo.save(ctx, tx, &r, false)
	})
}

// Remove 整体校验并软删除，提交后移除调度计划。
func (s *Service) Remove(ctx context.Context, p security.Principal, ids []model.ID) error {
	if !p.Admin() {
		return response.ErrPermission
	}
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return response.ErrInvalidParam
	}
	return s.change(ctx, func(tx *gorm.DB) error {
		for _, id := range ids {
			if _, err := s.repo.get(ctx, tx, id); err != nil {
				return httpx.DBError(err)
			}
		}
		return s.repo.remove(ctx, tx, ids, p.ID)
	})
}

// Run 手动执行允许暂停配置，仍须满足白名单、参数和并发门禁。
func (s *Service) Run(ctx context.Context, p security.Principal, id model.ID) (RunResult, error) {
	if !p.Admin() {
		return RunResult{}, response.ErrPermission
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready.Load() || s.stopping.Load() {
		return RunResult{}, response.ErrUnavailable
	}
	row, err := s.repo.get(ctx, nil, id)
	if err != nil {
		return RunResult{}, httpx.DBError(err)
	}
	state, err := s.engine.Run(spec(row))
	return RunResult{state}, err
}

// Stop 禁止管理入口继续投递，停止配置重试生产者。
func (s *Service) Stop() {
	s.stopping.Store(true)
	s.cancel()
	s.wg.Wait()
	s.mu.Lock()
	s.ready.Store(false)
	s.mu.Unlock()
}

// Next 与实际调度共享同一个六位 UTC 解析器。
func (s *Service) Next(expr string) ([]time.Time, error) {
	return scheduler.Next(expr, time.Now().UTC())
}
