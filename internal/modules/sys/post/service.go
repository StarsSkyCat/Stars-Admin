package post

import (
	"context"
	"regexp"
	"strings"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

type idSource interface{ Next() (int64, error) }
type gate interface {
	Change(context.Context, func(*gorm.DB) ([]security.Revocation, error)) error
}

// Users 提供岗位引用检查，避免岗位功能读取用户表。
type Users interface {
	HasPosts(context.Context, *gorm.DB, []model.ID) (bool, error)
}

// Service 管理全局岗位；管理入口仅 admin，引用可由已授权用户选择。
type Service struct {
	repo  repository
	ids   idSource
	gate  gate
	users Users
}

// New 创建岗位服务。
func New(db *gorm.DB, ids idSource, g gate, users Users) *Service {
	return &Service{repository{db}, ids, g, users}
}

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// Validate 确认岗位存在且有效。
func (s *Service) Validate(ctx context.Context, tx *gorm.DB, ids []model.ID) error {
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	for _, id := range ids {
		row, err := s.repo.get(ctx, tx, id)
		if err != nil {
			return httpx.DBError(err)
		}
		if row.Status != 1 {
			return response.ErrConflict
		}
	}
	return nil
}

// List 分页返回岗位，options 只返回启用项。
func (s *Service) List(ctx context.Context, q response.PageQuery, options bool) (response.Page[View], error) {
	if err := q.Normalize(); err != nil {
		return response.Page[View]{}, err
	}
	column, desc, err := q.Sort(map[string]string{"id": "id", "sort": "sort", "name": "name"})
	if err != nil {
		return response.Page[View]{}, err
	}
	rows, total, err := s.repo.list(ctx, q, options, column, desc)
	out := response.Page[View]{List: []View{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, r := range rows {
		out.List = append(out.List, view(r))
	}
	return out, err
}

// Get 查询岗位详情。
func (s *Service) Get(ctx context.Context, id model.ID) (View, error) {
	row, err := s.repo.get(ctx, nil, id)
	return view(row), httpx.DBError(err)
}

// Save 在安全协调事务内新增或编辑岗位。
func (s *Service) Save(ctx context.Context, p security.Principal, id model.ID, in Input) (View, error) {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	if !codePattern.MatchString(in.Code) || strings.TrimSpace(in.Name) == "" || len([]rune(in.Name)) > 64 || (in.Status != 1 && in.Status != 2) {
		return View{}, response.ErrInvalidParam
	}
	var row post
	err := s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		if id == 0 {
			n, e := s.ids.Next()
			if e != nil {
				return nil, e
			}
			row = post{ID: model.ID(n), CreatedBy: p.ID}
		} else {
			var e error
			row, e = s.repo.get(ctx, tx, id)
			if e != nil {
				return nil, httpx.DBError(e)
			}
		}
		row.Code = in.Code
		row.Name = in.Name
		row.Sort = in.Sort
		row.Status = in.Status
		row.Remark = in.Remark
		row.UpdatedBy = p.ID
		return nil, httpx.DBError(s.repo.save(ctx, tx, &row, id == 0))
	})
	return view(row), err
}

// Remove 拒绝仍被用户引用的岗位，批量整体回滚。
func (s *Service) Remove(ctx context.Context, p security.Principal, ids []model.ID) error {
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return response.ErrInvalidParam
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		for _, id := range ids {
			if _, e := s.repo.get(ctx, tx, id); e != nil {
				return nil, httpx.DBError(e)
			}
		}
		used, e := s.users.HasPosts(ctx, tx, ids)
		if e != nil {
			return nil, e
		}
		if used {
			return nil, response.ErrConflict
		}
		return nil, s.repo.remove(ctx, tx, ids, p.ID)
	})
}
