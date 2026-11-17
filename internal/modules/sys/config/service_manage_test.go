package config

import (
	"errors"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
)

func TestParameterValidationProtectsSecuritySettings(t *testing.T) {
	for _, in := range []Input{{Name: "参数", Key: "sys.account.captchaEnabled", Value: "1"}, {Name: "参数", Key: "sys.account.maxSessions", Value: "0"}, {Name: "参数", Key: "sys.account.lockMinutes", Value: "101"}, {Name: "参数", Key: "sys.account.unknown", Value: "true"}, {Name: "参数", Key: "service.secret", Value: "hidden"}, {Name: "参数", Key: "bad*key", Value: "x"}} {
		if err := validate(in); !errors.Is(err, response.ErrInvalidParam) {
			t.Fatalf("未拒绝非法参数键 %s", in.Key)
		}
	}
	for _, in := range []Input{{Name: "参数", Key: "sys.account.captchaEnabled", Value: "false"}, {Name: "参数", Key: "sys.account.passwordRetryLimit", Value: "5"}, {Name: "名称", Key: "app.title", Value: "Stars"}} {
		if err := validate(in); err != nil {
			t.Fatal(err)
		}
	}
}
