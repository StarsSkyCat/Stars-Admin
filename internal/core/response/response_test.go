package response

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWrappedErrorsAndUnknownFailure(t *testing.T) {
	for _, test := range []struct {
		err          error
		status, code int
	}{
		{fmt.Errorf("内部上下文: %w", ErrInvalidParam), 400, 1000},
		{errors.Join(errors.New("连接信息不能外泄"), ErrConflict), 409, 1003},
		{errors.New("password=secret SQL SELECT"), 500, 1500},
		{ErrUnavailable, 503, 1501},
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		Fail(c, test.err)
		var body Body[any]
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != test.status || body.Code != test.code || !c.IsAborted() {
			t.Fatalf("错误映射不正确：%d/%d", w.Code, body.Code)
		}
		if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "SQL") {
			t.Fatal("内部错误外泄")
		}
	}
}

func TestEmptyPageAndMarshalFailure(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	OK(c, Page[int]{Page: 1, PageSize: 20})
	if !strings.Contains(w.Body.String(), `"list":[]`) {
		t.Fatal("空列表必须为数组")
	}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	OK(c, make(chan int))
	if w.Code != 500 || !strings.Contains(w.Body.String(), `"code":1500`) {
		t.Fatal("序列化失败必须返回统一错误")
	}
}

func TestPaginationAndSortWhitelist(t *testing.T) {
	q := PageQuery{}
	if err := q.Normalize(); err != nil || q.Page != 1 || q.PageSize != 20 {
		t.Fatal("分页默认值错误")
	}
	for _, q := range []PageQuery{{Page: -1}, {PageSize: 101}, {PageSize: -1}} {
		if q.Normalize() == nil {
			t.Fatal("分页越界未拒绝")
		}
	}
	for _, input := range []string{"id; DROP TABLE sys_user", "password desc", "id sideways"} {
		if _, _, err := (PageQuery{OrderBy: input}).Sort(map[string]string{"id": "sys_user.id"}); err == nil {
			t.Fatal("非法排序未拒绝")
		}
	}
	column, desc, err := (PageQuery{OrderBy: "id DESC"}).Sort(map[string]string{"id": "sys_user.id"})
	if err != nil || column != "sys_user.id" || !desc {
		t.Fatal("合法排序映射错误")
	}
}
