//go:build integration

package permission

import (
	"context"
	"strings"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/testutil"
	"gorm.io/gorm"
)

func TestNamespaceIsolationAndTransactionalProjection(t *testing.T) {
	cfg, err := config.Load("../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	store, err := New(deps.DB, cfg.Database.Timeout)
	if err != nil {
		t.Fatal(err)
	}
	rules := []Rule{{Ptype: "p", V0: "role:55", V1: "sys:user:remove"}, {Ptype: "g", V0: "user:55", V1: "role:56"}, {Ptype: "p", V0: "role:56", V1: "sys:user:list"}, {Ptype: "p", V0: "role:56", V1: strings.Repeat("x", 128)}}
	if err := deps.DB.WithContext(t.Context()).Transaction(func(tx *gorm.DB) error { return store.Replace(t.Context(), tx, rules) }); err != nil {
		t.Fatal(err)
	}
	if err := store.Reload(t.Context()); err != nil {
		t.Fatal(err)
	}
	if allowed, err := store.Allowed(55, "sys:user:remove"); err != nil || allowed {
		t.Fatal("相同用户与角色 ID 导致串权")
	}
	if allowed, err := store.Allowed(55, "sys:user:list"); err != nil || !allowed {
		t.Fatal("正常角色继承失效")
	}
	if allowed, err := store.Allowed(55, strings.Repeat("x", 128)); err != nil || !allowed {
		t.Fatal("权限列宽不足")
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := store.Reload(cancelled); err == nil {
		t.Fatal("策略加载必须遵守调用方取消")
	}
	if allowed, err := store.Allowed(55, "sys:user:list"); err != nil || !allowed {
		t.Fatal("失败的候选策略不应替换已加载模型")
	}
	duplicate := Rule{Ptype: rules[0].Ptype, V0: rules[0].V0, V1: rules[0].V1}
	if err := deps.DB.Create(&duplicate).Error; err == nil {
		t.Fatal("策略唯一约束未生效")
	}
}
