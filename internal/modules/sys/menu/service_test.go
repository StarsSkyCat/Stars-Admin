package menu

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

func (directGate) Change(_ context.Context, fn func(*gorm.DB) ([]security.Revocation, error)) error {
	_, err := fn(nil)
	return err
}

type memoryMenus struct {
	repository
	rows map[model.ID]menu
}

func (r *memoryMenus) all(context.Context, *gorm.DB) ([]menu, error) {
	out := []menu{}
	for _, row := range r.rows {
		out = append(out, row)
	}
	return out, nil
}
func (r *memoryMenus) get(_ context.Context, _ *gorm.DB, id model.ID) (menu, error) {
	row, ok := r.rows[id]
	if !ok {
		return row, gorm.ErrRecordNotFound
	}
	return row, nil
}
func (r *memoryMenus) save(_ context.Context, _ *gorm.DB, row *menu, _ bool) error {
	r.rows[row.ID] = *row
	return nil
}

func TestMenuAncestorStateAndCycle(t *testing.T) {
	repo := &memoryMenus{rows: map[model.ID]menu{10: {ID: 10, Name: "父", Type: 1, Status: 2}, 11: {ID: 11, ParentID: 10, Name: "按钮", Type: 3, Status: 1, Perm: new("sys:user:add")}, 20: {ID: 20, Name: "有效", Type: 3, Status: 1, Perm: new("sys:user:list")}}}
	s := &Service{repo: repo, gate: directGate{}}
	perms, err := s.Permissions(t.Context(), nil)
	if err != nil || len(perms) != 1 || perms[20] != "sys:user:list" {
		t.Fatal("停用父级的权限不能生效")
	}
	if _, err := s.Save(t.Context(), security.Principal{}, 10, Input{ParentID: 11, Name: "父", Type: 1, Status: 1, Visible: 1}); !errors.Is(err, response.ErrConflict) {
		t.Fatal("菜单防环失效")
	}
	if _, err := s.Save(t.Context(), security.Principal{}, 10, Input{Name: "按钮", Type: 3, Perm: new("sys:user:*"), Status: 1, Visible: 1}); !errors.Is(err, response.ErrInvalidParam) {
		t.Fatal("权限通配符未拒绝")
	}
}
