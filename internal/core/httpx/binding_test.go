package httpx

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/gin-gonic/gin"
)

func TestStrictBindingAndIDLists(t *testing.T) {
	for _, body := range []string{`{"name":"x","roleIds":["1"]}`, `{"name":"x","name":"y"}`, strings.Repeat("x", (1<<20)+1)} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader(body))
		var in struct {
			Name string `json:"name"`
		}
		if Bind(c, &in) == nil {
			t.Fatal("未知、重复或过大字段未拒绝")
		}
	}
	for _, text := range []string{"0", "01", "-1", "9223372036854775808", "1,1", "1,", "", "1," + strings.Repeat("2,", 100)} {
		if _, err := IDs(text); err == nil {
			t.Fatal("非法 ID 集合未拒绝")
		}
	}
	if ids, err := IDs("1,9223372036854775807"); err != nil || len(ids) != 2 {
		t.Fatal("合法 ID 集合被拒绝")
	}
	if UniqueIDs([]model.ID{1, 1}) == nil || UniqueIDs([]model.ID{}) != nil {
		t.Fatal("显式解绑或重复校验错误")
	}
}
