package config

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/cache"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

// Configure 接入普通查询缓存；认证 Read 始终从数据库读取。
func (s *Service) Configure(c *cache.Store, ids idSource) { s.cache = c; s.ids = ids }

var keyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.-]{0,127}$`)

func validate(in Input) error {
	if !keyPattern.MatchString(in.Key) || strings.TrimSpace(in.Name) == "" || len(in.Value) > 4096 {
		return response.ErrInvalidParam
	}
	lower := strings.ToLower(in.Key)
	for _, word := range []string{"password", "secret", "token", "credential", "privatekey"} {
		if strings.Contains(lower, word) {
			if in.Key != "sys.account.passwordRetryLimit" {
				return response.ErrInvalidParam
			}
		}
	}
	if strings.HasPrefix(in.Key, "sys.account.") {
		switch in.Key {
		case "sys.account.captchaEnabled", "sys.account.multiLogin":
			if in.Value != "true" && in.Value != "false" {
				return response.ErrInvalidParam
			}
		case "sys.account.maxSessions", "sys.account.passwordRetryLimit", "sys.account.lockMinutes":
			n, err := strconv.Atoi(in.Value)
			if err != nil || n < 1 || n > 100 {
				return response.ErrInvalidParam
			}
		default:
			return response.ErrInvalidParam
		}
	}
	return nil
}

// List 返回参数管理分页。
func (s *Service) List(ctx context.Context, q response.PageQuery) (response.Page[View], error) {
	if err := q.Normalize(); err != nil {
		return response.Page[View]{}, err
	}
	if q.OrderBy != "" && q.OrderBy != "id" {
		return response.Page[View]{}, response.ErrInvalidParam
	}
	rows, total, err := s.repo.list(ctx, q)
	out := response.Page[View]{List: []View{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, r := range rows {
		out.List = append(out.List, view(r))
	}
	return out, err
}

// Get 返回参数详情。
func (s *Service) Get(ctx context.Context, id model.ID) (View, error) {
	row, err := s.repo.get(ctx, nil, id)
	return view(row), httpx.DBError(err)
}

// ByKey 只服务于经过 get 权限校验的管理入口。
func (s *Service) ByKey(ctx context.Context, key string) (View, error) {
	if !keyPattern.MatchString(key) {
		return View{}, response.ErrInvalidParam
	}
	var out View
	if s.cache != nil && s.cache.Read(ctx, "config:"+key, &out) {
		return out, nil
	}
	row, err := s.repo.byKey(ctx, key)
	if err != nil {
		return out, httpx.DBError(err)
	}
	out = view(row)
	if s.cache != nil {
		s.cache.Put(ctx, "config:"+key, out)
	}
	return out, nil
}

// Save 保持参数键不可改名，提交后失效缓存。
func (s *Service) Save(ctx context.Context, p security.Principal, id model.ID, in Input) (View, error) {
	if err := validate(in); err != nil {
		return View{}, err
	}
	var row parameter
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id == 0 {
			n, e := s.ids.Next()
			if e != nil {
				return e
			}
			row = parameter{ID: model.ID(n), CreatedBy: p.ID, Key: in.Key}
			if strings.HasPrefix(in.Key, "sys.account.") {
				return response.ErrPermission
			}
		} else {
			var e error
			row, e = s.repo.get(ctx, tx, id)
			if e != nil {
				return httpx.DBError(e)
			}
			if row.Key != in.Key {
				return response.ErrInvalidParam
			}
		}
		row.Name = in.Name
		row.Value = in.Value
		row.Remark = in.Remark
		row.UpdatedBy = p.ID
		return httpx.DBError(s.repo.save(ctx, tx, &row, id == 0))
	})
	if err == nil && s.cache != nil {
		s.cache.Invalidate("config:" + in.Key)
	}
	return view(row), err
}

// Remove 内置参数不可删除，批量操作整体回滚。
func (s *Service) Remove(ctx context.Context, _ security.Principal, ids []model.ID) error {
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return response.ErrInvalidParam
	}
	keys := []string{}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			r, e := s.repo.get(ctx, tx, id)
			if e != nil {
				return httpx.DBError(e)
			}
			if r.IsSystem || strings.HasPrefix(r.Key, "sys.account.") {
				return response.ErrPermission
			}
			keys = append(keys, r.Key)
		}
		return s.repo.remove(ctx, tx, ids)
	})
	if err == nil && s.cache != nil {
		for _, key := range keys {
			s.cache.Invalidate("config:" + key)
		}
	}
	return err
}

// Refresh 显式失效所有参数缓存。
func (s *Service) Refresh(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	if err := s.cache.Refresh(ctx, "config:*"); err != nil {
		return response.ErrUnavailable
	}
	return nil
}
