package post

import (
	"errors"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
)

func TestPostValidationBeforePersistence(t *testing.T) {
	s := new(Service)
	if _, err := s.Save(t.Context(), security.Principal{}, 0, Input{Code: "bad*", Name: "岗位", Status: 1}); !errors.Is(err, response.ErrInvalidParam) {
		t.Fatal("非法编码未拒绝")
	}
	if err := s.Validate(t.Context(), nil, []model.ID{2, 2}); !errors.Is(err, response.ErrInvalidParam) {
		t.Fatal("重复岗位引用未拒绝")
	}
}
