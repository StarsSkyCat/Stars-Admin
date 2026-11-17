package dict

import (
	"errors"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
)

func TestDictionaryRejectsInvalidIdentifiersBeforePersistence(t *testing.T) {
	s := new(Service)
	for _, key := range []string{"", "a*", "bad-key", "1type"} {
		if _, err := s.SaveType(t.Context(), security.Principal{ID: 1}, 0, TypeInput{Name: "字典", Type: key, Status: 1}); !errors.Is(err, response.ErrInvalidParam) {
			t.Fatal("非法类型应在访问数据库前拒绝")
		}
	}
	if _, err := s.SaveData(t.Context(), security.Principal{ID: 1}, 0, DataInput{DictType: "valid", Label: "数据", Value: "x", Status: 3}); !errors.Is(err, response.ErrInvalidParam) {
		t.Fatal("非法状态应拒绝")
	}
}
