package dept

import (
	"context"
	"errors"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

type directGate struct{}

func (directGate) Update(_ context.Context, fn func(*gorm.DB) ([]security.Revocation, error)) error {
	_, err := fn(nil)
	return err
}

func (directGate) Read(fn func() error) error { return fn() }

func (directGate) Allow(security.Principal, string) error { return nil }

func (directGate) Resolve(context.Context, *gorm.DB, security.Principal) (security.Scope, error) {
	return security.Scope{All: true}, nil
}

type memoryDepartments struct {
	repository
	rows       map[model.ID]department
	saved      []model.ID
	lastQuery  Query
	lastColumn string
}

func (r *memoryDepartments) list(_ context.Context, q Query, column string, desc bool) ([]department, int64, error) {
	r.lastQuery = q
	r.lastColumn = column
	if desc {
		return nil, 0, errors.New("测试仓储期望升序")
	}
	return []department{r.rows[2]}, 2, nil
}

func (r *memoryDepartments) all(context.Context, *gorm.DB) ([]department, error) {
	out := []department{}
	for _, row := range r.rows {
		out = append(out, row)
	}
	return out, nil
}
func (r *memoryDepartments) get(_ context.Context, _ *gorm.DB, id model.ID) (department, error) {
	row, ok := r.rows[id]
	if !ok {
		return row, gorm.ErrRecordNotFound
	}
	return row, nil
}
func (r *memoryDepartments) save(_ context.Context, _ *gorm.DB, row *department, _ bool) error {
	r.rows[row.ID] = *row
	r.saved = append(r.saved, row.ID)
	return nil
}

func TestMoveSubtreeAndCycleProtection(t *testing.T) {
	repo := &memoryDepartments{rows: map[model.ID]department{10: {ID: 10, Name: "A", Ancestors: "0", Status: 1}, 20: {ID: 20, Name: "B", Ancestors: "0", Status: 1}, 11: {ID: 11, ParentID: 10, Name: "子", Ancestors: "0,10", Status: 1}, 100: {ID: 100, Name: "边界", Ancestors: "0", Status: 1}, 101: {ID: 101, ParentID: 100, Name: "非后代", Ancestors: "0,100", Status: 1}}}
	gate := directGate{}
	s := &Service{repo: repo, gate: gate, access: security.NewScopeAccess(gate, gate)}
	if _, err := s.Save(t.Context(), security.Principal{ID: 1}, 10, Input{ParentID: 11, Name: "A", Status: 1}); !errors.Is(err, response.ErrConflict) || len(repo.saved) != 0 {
		t.Fatal("后代父级必须在写入前拒绝")
	}
	if _, err := s.Save(t.Context(), security.Principal{ID: 1}, 10, Input{ParentID: 20, Name: "A", Status: 1}); err != nil {
		t.Fatal(err)
	}
	if repo.rows[10].Ancestors != "0,20" || repo.rows[11].Ancestors != "0,20,10" || repo.rows[101].Ancestors != "0,100" {
		t.Fatal("祖级更新没有遵守逗号边界")
	}
}

func TestDepartmentPaginationAndSortValidation(t *testing.T) {
	repo := &memoryDepartments{rows: map[model.ID]department{1: {ID: 1, Name: "B"}, 2: {ID: 2, Name: "A"}}}
	gate := directGate{}
	s := &Service{repo: repo, access: security.NewScopeAccess(gate, gate)}
	ctx := security.WithPrincipal(t.Context(), security.Principal{ID: 1})
	q := Query{}
	q.PageSize = 1
	q.OrderBy = "name asc"
	out, err := s.List(ctx, q)
	if err != nil || out.Total != 2 || len(out.List) != 1 || out.List[0].Name != "A" {
		t.Fatal("分页或排序错误")
	}
	if repo.lastQuery.Page != 1 || repo.lastQuery.PageSize != 1 || repo.lastColumn != "name" {
		t.Fatal("未将标准化页码与可信列名交给仓储")
	}
	q.OrderBy = "name;DROP TABLE"
	if _, err := s.List(ctx, q); err == nil {
		t.Fatal("非法排序未拒绝")
	}
}
