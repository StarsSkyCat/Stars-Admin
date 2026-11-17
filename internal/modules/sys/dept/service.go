package dept

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

type idSource interface{ Next() (int64, error) }
type changeGate interface {
	Change(context.Context, func(*gorm.DB) ([]security.Revocation, error)) error
}

type departmentStore interface {
	list(context.Context, Query, string, bool) ([]department, int64, error)
	all(context.Context, *gorm.DB) ([]department, error)
	get(context.Context, *gorm.DB, model.ID) (department, error)
	save(context.Context, *gorm.DB, *department, bool) error
	remove(context.Context, *gorm.DB, []model.ID, model.ID) error
	seedRoot(context.Context, *gorm.DB) (department, bool, error)
}

// Users 由装配层注入用户业务的引用检查，不读取其他功能的数据表。
type Users interface {
	HasDepartments(context.Context, *gorm.DB, []model.ID) (bool, error)
}

// Service 维护部门层级，不在 handler 暴露持久化对象。
type Service struct {
	repo   departmentStore
	ids    idSource
	gate   changeGate
	users  Users
	scopes security.Resolver
}

// New 构造部门业务。
func New(db *gorm.DB, ids idSource, gate changeGate) *Service {
	return &Service{repo: repository{db}, ids: ids, gate: gate}
}

// SetUsers 在功能装配完成后连接引用检查接口。
func (s *Service) SetUsers(users Users) { s.users = users }

// Exists 校验关联部门必须有效且启用。
func (s *Service) Exists(ctx context.Context, tx *gorm.DB, id model.ID) error {
	row, err := s.repo.get(ctx, tx, id)
	if err != nil {
		return httpx.DBError(err)
	}
	if row.Status != 1 {
		return response.ErrConflict
	}
	return nil
}

// List 返回分页部门；排序字段只允许既有业务列。
func (s *Service) list(ctx context.Context, q Query) (response.Page[View], error) {
	var scopeErr error
	ctx, scopeErr = s.scopeContext(ctx, nil, security.Actor(ctx))
	if scopeErr != nil {
		return response.Page[View]{}, scopeErr
	}
	if err := q.Normalize(); err != nil {
		return response.Page[View]{}, err
	}
	column, desc, err := q.Sort(map[string]string{"id": "id", "sort": "sort", "name": "name"})
	if err != nil {
		return response.Page[View]{}, err
	}
	rows, total, err := s.repo.list(ctx, q, column, desc)
	if err != nil {
		return response.Page[View]{}, err
	}
	result := response.Page[View]{List: []View{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, row := range rows {
		result.List = append(result.List, view(row))
	}
	return result, nil
}

// Get 返回单个部门。
func (s *Service) get(ctx context.Context, id model.ID) (View, error) {
	var scopeErr error
	ctx, scopeErr = s.scopeContext(ctx, nil, security.Actor(ctx))
	if scopeErr != nil {
		return View{}, scopeErr
	}
	row, err := s.repo.get(ctx, nil, id)
	return view(row), httpx.DBError(err)
}

// Tree 返回按显示顺序排列的完整部门树。
func (s *Service) tree(ctx context.Context) ([]View, error) {
	var scopeErr error
	ctx, scopeErr = s.scopeContext(ctx, nil, security.Actor(ctx))
	if scopeErr != nil {
		return nil, scopeErr
	}
	rows, err := s.repo.all(ctx, nil)
	if err != nil {
		return nil, err
	}
	return scopedTree(ctx, rows), nil
}

// Save 在一个事务中校验父链并更新整个子树路径。
func (s *Service) Save(ctx context.Context, p security.Principal, id model.ID, in Input) (View, error) {
	var out department
	err := s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		perm := "sys:dept:edit"
		if id == 0 {
			perm = "sys:dept:add"
		}
		var scopeErr error
		ctx, scopeErr = s.scopeContext(ctx, tx, p, perm)
		if scopeErr != nil {
			return nil, scopeErr
		}
		rows, err := s.repo.all(ctx, tx)
		if err != nil {
			return nil, err
		}
		if in.ParentID < 0 || strings.TrimSpace(in.Name) == "" || utf8.RuneCountInString(in.Name) > 64 || in.Status < 1 || in.Status > 2 {
			return nil, response.ErrInvalidParam
		}
		if id == 0 {
			n, err := s.ids.Next()
			if err != nil {
				return nil, err
			}
			out = department{ID: model.ID(n), CreatedBy: p.ID}
		} else {
			out, err = s.repo.get(ctx, tx, id)
			if err != nil {
				return nil, httpx.DBError(err)
			}
		}
		if in.ParentID == 0 && (id == 0 || out.ParentID != 0) {
			if scope, ok := security.CurrentScope(ctx); ok && !scope.All {
				return nil, response.ErrPermission
			}
		}
		ancestors := "0"
		if id != 0 {
			ancestors = out.Ancestors
		}
		if in.ParentID != 0 && (id == 0 || out.ParentID != in.ParentID) {
			parent, err := s.repo.get(ctx, tx, in.ParentID)
			if err != nil {
				return nil, httpx.DBError(err)
			}
			if parent.Status != 1 {
				return nil, response.ErrConflict
			}
			if parent.ID == out.ID || strings.Contains(","+parent.Ancestors+",", ","+strconv.FormatInt(int64(out.ID), 10)+",") {
				return nil, response.ErrConflict
			}
			ancestors = parent.Ancestors + "," + strconv.FormatInt(int64(parent.ID), 10)
		}
		if in.ParentID == 0 {
			ancestors = "0"
		}
		if len(ancestors) > 500 {
			return nil, response.ErrInvalidParam
		}
		oldPrefix := out.Ancestors + "," + strconv.FormatInt(int64(out.ID), 10)
		out.ParentID = in.ParentID
		out.Ancestors = ancestors
		out.Name = strings.TrimSpace(in.Name)
		out.Sort = in.Sort
		out.Leader = in.Leader
		out.Phone = in.Phone
		out.Status = in.Status
		out.Remark = in.Remark
		out.UpdatedBy = p.ID
		if err := s.repo.save(ctx, tx, &out, id == 0); err != nil {
			return nil, httpx.DBError(err)
		}
		if id != 0 && oldPrefix != ancestors+","+strconv.FormatInt(int64(out.ID), 10) {
			newPrefix := ancestors + "," + strconv.FormatInt(int64(out.ID), 10)
			for _, child := range rows {
				if child.Ancestors == oldPrefix || strings.HasPrefix(child.Ancestors, oldPrefix+",") {
					if err := checkDepartment(ctx, child.ID); err != nil {
						return nil, err
					}
					child.Ancestors = newPrefix + strings.TrimPrefix(child.Ancestors, oldPrefix)
					child.UpdatedBy = p.ID
					if len(child.Ancestors) > 500 {
						return nil, response.ErrInvalidParam
					}
					if err := s.repo.save(ctx, tx, &child, false); err != nil {
						return nil, err
					}
				}
			}
		}
		return nil, nil
	})
	return view(out), err
}

// Remove 有子部门或关联用户时拒绝删除，批量请求整体回滚。
func (s *Service) Remove(ctx context.Context, p security.Principal, ids []model.ID) error {
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return response.ErrInvalidParam
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		var scopeErr error
		ctx, scopeErr = s.scopeContext(ctx, tx, p, "sys:dept:remove")
		if scopeErr != nil {
			return nil, scopeErr
		}
		rows, err := s.repo.all(ctx, tx)
		if err != nil {
			return nil, err
		}
		targets := map[model.ID]bool{}
		for _, id := range ids {
			targets[id] = true
			if _, err := s.repo.get(ctx, tx, id); err != nil {
				return nil, httpx.DBError(err)
			}
		}
		for _, row := range rows {
			if targets[row.ParentID] {
				return nil, response.ErrConflict
			}
		}
		used, err := s.users.HasDepartments(ctx, tx, ids)
		if err != nil {
			return nil, err
		}
		if used {
			return nil, response.ErrConflict
		}
		return nil, s.repo.remove(ctx, tx, ids, p.ID)
	})
}
