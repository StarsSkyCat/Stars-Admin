//go:build integration

package config

import (
	"testing"

	coreconfig "github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/database"
	"github.com/StarsSkyCat/Stars-Admin/internal/testutil"
)

func TestSettingsDefaultsAndStrictTypes(t *testing.T) {
	cfg, err := coreconfig.Load("../../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	service := New(deps.DB)
	keys := []string{"sys.account.captchaEnabled", "sys.account.multiLogin", "sys.account.maxSessions", "sys.account.passwordRetryLimit", "sys.account.lockMinutes"}
	if err := deps.DB.Exec("DELETE FROM sys_config WHERE key IN ?", keys).Error; err != nil {
		t.Fatal(err)
	}
	settings, err := service.Read(t.Context())
	if err != nil || !settings.CaptchaEnabled || !settings.MultiLogin || settings.MaxSessions != 5 || settings.PasswordRetryLimit != 5 || settings.LockMinutes != 10 {
		t.Fatal("缺失参数的安全默认值错误")
	}
	if err := database.Migrate(t.Context(), cfg, "down", ""); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(t.Context(), cfg, "up", ""); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ key, value string }{{keys[0], "yes"}, {keys[2], "0"}, {keys[3], "101"}, {keys[4], "abc"}} {
		if err := deps.DB.Exec("UPDATE sys_config SET value=? WHERE key=?", test.value, test.key).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := service.Read(t.Context()); err == nil {
			t.Fatal("非法认证参数未拒绝")
		}
		value := "5"
		if test.key == keys[0] {
			value = "true"
		}
		if test.key == keys[4] {
			value = "10"
		}
		if err := deps.DB.Exec("UPDATE sys_config SET value=? WHERE key=?", value, test.key).Error; err != nil {
			t.Fatal(err)
		}
	}
}
