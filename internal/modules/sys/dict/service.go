package dict

import (
	"context"
	"regexp"
	"strings"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/cache"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

// Service 管理类型、数据和普通缓存。
type Service struct {
	repo  repository
	ids   idSource
	cache *cache.Store
}

// New 创建字典业务。
func New(db *gorm.DB, ids idSource, c *cache.Store) *Service { return &Service{repository{db}, ids, c} }

var typePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// ListTypes 查询字典类型分页。
func (s *Service) ListTypes(ctx context.Context, q response.PageQuery) (response.Page[TypeView], error) {
	if err := q.Normalize(); err != nil {
		return response.Page[TypeView]{}, err
	}
	if q.OrderBy != "" && q.OrderBy != "id" {
		return response.Page[TypeView]{}, response.ErrInvalidParam
	}
	rows, _, total, err := s.repo.list(ctx, q, false, "")
	out := response.Page[TypeView]{List: []TypeView{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, r := range rows {
		out.List = append(out.List, typeView(r))
	}
	return out, err
}

// ListData 查询字典项分页。
func (s *Service) ListData(ctx context.Context, q response.PageQuery, key string) (response.Page[DataView], error) {
	if err := q.Normalize(); err != nil {
		return response.Page[DataView]{}, err
	}
	if q.OrderBy != "" && q.OrderBy != "id" {
		return response.Page[DataView]{}, response.ErrInvalidParam
	}
	_, rows, total, err := s.repo.list(ctx, q, true, key)
	out := response.Page[DataView]{List: []DataView{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, r := range rows {
		out.List = append(out.List, dataView(r))
	}
	return out, err
}

// GetType 查询类型详情。
func (s *Service) GetType(ctx context.Context, id model.ID) (TypeView, error) {
	r, e := s.repo.typeGet(ctx, nil, id)
	return typeView(r), httpx.DBError(e)
}

// GetData 查询字典项详情。
func (s *Service) GetData(ctx context.Context, id model.ID) (DataView, error) {
	r, e := s.repo.dataGet(ctx, nil, id)
	return dataView(r), httpx.DBError(e)
}

// ByType 为已登录主体读取启用字典项。
func (s *Service) ByType(ctx context.Context, key string) ([]DataView, error) {
	if !typePattern.MatchString(key) {
		return nil, response.ErrInvalidParam
	}
	out := []DataView{}
	if s.cache.Read(ctx, "dict:"+key, &out) {
		return out, nil
	}
	typ, err := s.repo.typeKey(ctx, nil, key)
	if err != nil {
		return nil, httpx.DBError(err)
	}
	if typ.Status == 1 {
		rows, err := s.repo.enabled(ctx, key)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, dataView(r))
		}
	}
	s.cache.Put(ctx, "dict:"+key, out)
	return out, nil
}

// SaveType 保证类型标识不可改名。
func (s *Service) SaveType(ctx context.Context, p security.Principal, id model.ID, in TypeInput) (TypeView, error) {
	in.Type = strings.ToLower(strings.TrimSpace(in.Type))
	if !typePattern.MatchString(in.Type) || strings.TrimSpace(in.Name) == "" || (in.Status != 1 && in.Status != 2) {
		return TypeView{}, response.ErrInvalidParam
	}
	var row dictionaryType
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := s.repo.lock(ctx, tx); e != nil {
			return e
		}
		if id == 0 {
			n, e := s.ids.Next()
			if e != nil {
				return e
			}
			row = dictionaryType{ID: model.ID(n), CreatedBy: p.ID, Type: in.Type}
		} else {
			var e error
			row, e = s.repo.typeGet(ctx, tx, id)
			if e != nil {
				return httpx.DBError(e)
			}
			if row.Type != in.Type {
				return response.ErrInvalidParam
			}
		}
		row.Name = in.Name
		row.Status = in.Status
		row.Remark = in.Remark
		row.UpdatedBy = p.ID
		return httpx.DBError(s.repo.save(ctx, tx, &row, id == 0))
	})
	if err == nil {
		s.cache.Invalidate("dict:" + in.Type)
	}
	return typeView(row), err
}

// SaveData 同事务校验类型引用并维护缓存。
func (s *Service) SaveData(ctx context.Context, p security.Principal, id model.ID, in DataInput) (DataView, error) {
	if !typePattern.MatchString(in.DictType) || strings.TrimSpace(in.Label) == "" || in.Value == "" || (in.Status != 1 && in.Status != 2) {
		return DataView{}, response.ErrInvalidParam
	}
	var row dictionaryData
	old := ""
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := s.repo.lock(ctx, tx); e != nil {
			return e
		}
		typ, e := s.repo.typeKey(ctx, tx, in.DictType)
		if e != nil {
			return httpx.DBError(e)
		}
		if typ.Status != 1 {
			return response.ErrConflict
		}
		if id == 0 {
			n, e := s.ids.Next()
			if e != nil {
				return e
			}
			row = dictionaryData{ID: model.ID(n), CreatedBy: p.ID}
		} else {
			row, e = s.repo.dataGet(ctx, tx, id)
			if e != nil {
				return httpx.DBError(e)
			}
			old = row.DictType
		}
		row.DictType = in.DictType
		row.Label = in.Label
		row.Value = in.Value
		row.Sort = in.Sort
		row.Tag = in.Tag
		row.IsDefault = in.IsDefault
		row.Status = in.Status
		row.Remark = in.Remark
		row.UpdatedBy = p.ID
		return httpx.DBError(s.repo.save(ctx, tx, &row, id == 0))
	})
	if err == nil {
		s.cache.Invalidate("dict:" + in.DictType)
		if old != "" && old != in.DictType {
			s.cache.Invalidate("dict:" + old)
		}
	}
	return dataView(row), err
}

// Remove 批量删除在同一事务中执行，类型删除同时软删除关联项。
func (s *Service) Remove(ctx context.Context, p security.Principal, ids []model.ID, data bool) error {
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return response.ErrInvalidParam
	}
	keys := []string{}
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := s.repo.lock(ctx, tx); e != nil {
			return e
		}
		for _, id := range ids {
			if data {
				r, e := s.repo.dataGet(ctx, tx, id)
				if e != nil {
					return httpx.DBError(e)
				}
				keys = append(keys, r.DictType)
			} else {
				r, e := s.repo.typeGet(ctx, tx, id)
				if e != nil {
					return httpx.DBError(e)
				}
				keys = append(keys, r.Type)
			}
		}
		if data {
			return s.repo.removeData(ctx, tx, ids, p.ID)
		}
		return s.repo.removeType(ctx, tx, ids, keys, p.ID)
	})
	if err == nil {
		for _, key := range keys {
			s.cache.Invalidate("dict:" + key)
		}
	}
	return err
}

// Refresh 显式清理字典缓存。
func (s *Service) Refresh(ctx context.Context) error {
	if err := s.cache.Refresh(ctx, "dict:*"); err != nil {
		return response.ErrUnavailable
	}
	return nil
}
