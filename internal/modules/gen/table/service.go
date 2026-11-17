package table

import (
	"context"
	"encoding/json/v2"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

type idSource interface{ Next() (int64, error) }

// Service 管理元数据事务、生成配置和输出，不跨功能访问业务表。
type Service struct {
	repo       repository
	ids        idSource
	cfg        config.Gen
	env        string
	modulePath string
	mu         sync.Mutex
}

// New 创建生成服务，模块路径必须来自实际工程。
func New(db *gorm.DB, ids idSource, cfg config.Gen, env, modulePath string) *Service {
	return &Service{repo: repository{db}, ids: ids, cfg: cfg, env: env, modulePath: modulePath}
}

var dbName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

func (s *Service) allowed(name string) bool {
	if s.cfg.Schema != "public" || !dbName.MatchString(name) || name == "casbin_rule" || name == "schema_migrations" {
		return false
	}
	for _, prefix := range []string{"sys_", "job_", "gen_", "pg_", "sql_"} {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	for _, prefix := range s.cfg.AllowedPrefixes {
		if prefix != "" && strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// DBTables 过滤仅允许导入的 public 普通业务表。
func (s *Service) DBTables(ctx context.Context) ([]DBTable, error) {
	rows, err := s.repo.dbTables(ctx, s.cfg.Schema)
	out := []DBTable{}
	for _, r := range rows {
		if s.allowed(r.Name) {
			out = append(out, r)
		}
	}
	return out, err
}

// Import 整批获取结构并保存，任何表越界或重复都整体回滚。
func (s *Service) Import(ctx context.Context, p security.Principal, names []string) ([]View, error) {
	if !p.Admin() {
		return nil, response.ErrPermission
	}
	if len(names) == 0 || len(names) > 100 {
		return nil, response.ErrInvalidParam
	}
	seen := map[string]bool{}
	for _, name := range names {
		if !s.allowed(name) || seen[name] {
			return nil, response.ErrInvalidParam
		}
		seen[name] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []View{}
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, name := range names {
			snapshot, err := s.repo.describe(ctx, tx, s.cfg.Schema, name)
			if err != nil {
				return err
			}
			if len(snapshot.Columns) > 128 {
				return response.ErrInvalidParam
			}
			n, err := s.ids.Next()
			if err != nil {
				return err
			}
			feature := name
			for _, prefix := range s.cfg.AllowedPrefixes {
				if strings.HasPrefix(feature, prefix) {
					feature = strings.TrimPrefix(feature, prefix)
					break
				}
			}
			raw, err := json.Marshal(options{Snapshot: snapshot, Scope: ScopeConfig{Mode: "deny"}})
			if err != nil {
				return err
			}
			row := table{ID: model.ID(n), CreatedBy: p.ID, UpdatedBy: p.ID, TableNameValue: name, TableComment: truncate(snapshot.Comment, 255), ModuleName: "business", FeatureName: feature, BusinessName: feature, GenType: "zip", Options: string(raw), SyncedAt: new(time.Now().UTC())}
			if err := s.repo.save(ctx, tx, &row, true); err != nil {
				return httpx.DBError(err)
			}
			cols, err := s.merge(ctx, tx, p, row.ID, nil, snapshot.Columns)
			if err != nil {
				return err
			}
			out = append(out, view(row, cols))
		}
		return nil
	})
	return out, err
}
func truncate(v string, n int) string { r := []rune(v); return string(r[:min(len(r), n)]) }
func view(r table, cols []column) View {
	var opts options
	_ = json.Unmarshal([]byte(r.Options), &opts)
	out := View{ID: r.ID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, TableName: r.TableNameValue, TableComment: r.TableComment, ModuleName: r.ModuleName, FeatureName: r.FeatureName, BusinessName: r.BusinessName, Author: r.Author, GenType: r.GenType, Scope: opts.Scope, Columns: []ColumnView{}, SyncedAt: r.SyncedAt}
	for _, c := range cols {
		out.Columns = append(out.Columns, ColumnView{ID: c.ID, ColumnInput: ColumnInput{c.ColumnName, c.GoField, c.JSONField, c.IsRequired, c.IsInsert, c.IsEdit, c.IsList, c.IsQuery, c.QueryType, c.HTMLType, c.DictType}, DBType: c.DBType, GoType: c.GoType, IsPK: c.IsPK, IsNullable: c.IsNullable, MaxLength: c.MaxLength, NumericPrecision: c.NumericPrecision, NumericScale: c.NumericScale, ColumnDefault: c.ColumnDefault, ChangeKind: c.ChangeKind, Removed: c.DeletedAt.Valid})
	}
	return out
}

// Get 返回配置和同步变化标记。
func (s *Service) Get(ctx context.Context, id model.ID) (View, error) {
	r, err := s.repo.get(ctx, nil, id)
	if err != nil {
		return View{}, httpx.DBError(err)
	}
	cols, err := s.repo.columns(ctx, nil, id, true)
	return view(r, cols), err
}

// List 返回已导入表分页。
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
		out.List = append(out.List, view(r, nil))
	}
	return out, err
}

// Sync 按列名保留人工配置，删除列不生成，类型或主键变化后必须重新配置。
func (s *Service) Sync(ctx context.Context, p security.Principal, id model.ID) (View, error) {
	if !p.Admin() {
		return View{}, response.ErrPermission
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var result View
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		r, err := s.repo.get(ctx, tx, id)
		if err != nil {
			return httpx.DBError(err)
		}
		if !s.allowed(r.TableNameValue) {
			return response.ErrPermission
		}
		var opts options
		if err := json.Unmarshal([]byte(r.Options), &opts); err != nil {
			return err
		}
		snap, err := s.repo.describe(ctx, tx, s.cfg.Schema, r.TableNameValue)
		if err != nil {
			return err
		}
		if len(snap.Columns) > 128 {
			return response.ErrInvalidParam
		}
		old, err := s.repo.columns(ctx, tx, id, true)
		if err != nil {
			return err
		}
		cols, err := s.merge(ctx, tx, p, id, old, snap.Columns)
		if err != nil {
			return err
		}
		opts.Snapshot = snap
		opts.Configured = false
		raw, err := json.Marshal(opts)
		if err != nil {
			return err
		}
		r.Options = string(raw)
		r.TableComment = truncate(snap.Comment, 255)
		r.SyncedAt = new(time.Now().UTC())
		r.UpdatedBy = p.ID
		if err := s.repo.save(ctx, tx, &r, false); err != nil {
			return err
		}
		result = view(r, cols)
		return nil
	})
	return result, err
}

// Remove 仅删除生成配置，不删除原业务表。
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
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			if _, err := s.repo.get(ctx, tx, id); err != nil {
				return httpx.DBError(err)
			}
		}
		return s.repo.remove(ctx, tx, ids, p.ID)
	})
}
func common(name string) bool {
	return slices.Contains([]string{"id", "created_by", "updated_by", "created_at", "updated_at", "deleted_at", "remark"}, name)
}
