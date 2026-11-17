// Package response 定义统一的 JSON 响应、错误码与分页结构。
package response

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Body 是所有 JSON API 的统一响应体。
type Body[T any] struct {
	Code int    `json:"code" example:"0"`
	Msg  string `json:"msg" example:"ok"`
	Data T      `json:"data"`
}

// Status 是存活、就绪和连通检查的公开响应数据。
type Status struct {
	Status string `json:"status" example:"ok"`
}

// Page 是分页响应，空列表编码为数组。
type Page[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

// BizError 只携带允许公开的业务文案，不包含内部原因。
type BizError struct {
	Code int
	Msg  string
}

func (e *BizError) Error() string { return e.Msg }

var (
	ErrInvalidParam    = &BizError{1000, "参数错误"}
	ErrNotFound        = &BizError{1001, "资源不存在"}
	ErrDuplicateSubmit = &BizError{1002, "重复提交"}
	ErrConflict        = &BizError{1003, "资源状态或唯一性冲突"}
	ErrRateLimited     = &BizError{1004, "请求过于频繁"}
	ErrForbidden       = &BizError{1005, "访问被拒绝"}
	ErrMethod          = &BizError{1006, "请求方法不受支持"}
	ErrInternal        = &BizError{1500, "服务内部错误"}
	ErrUnavailable     = &BizError{1501, "服务暂不可用"}
	ErrUnauthenticated = &BizError{2000, "请先登录"}
	ErrTokenExpired    = &BizError{2001, "令牌已过期"}
	ErrCredentials     = &BizError{2002, "用户名或密码错误"}
	ErrAccountLocked   = &BizError{2003, "账号暂时锁定"}
	ErrPermission      = &BizError{2004, "没有操作权限"}
	ErrCaptcha         = &BizError{2005, "验证码错误或已过期"}
	ErrAccountDisabled = &BizError{2006, "账号或角色已停用"}
	ErrRefresh         = &BizError{2007, "刷新令牌失效"}
	ErrPasswordChange  = &BizError{2008, "首次登录或密码重置后须修改密码"}
)

var statuses = map[int]int{
	1000: http.StatusBadRequest, 1001: http.StatusNotFound,
	1002: http.StatusConflict, 1003: http.StatusConflict, 1004: http.StatusTooManyRequests,
	1005: http.StatusForbidden, 1006: http.StatusMethodNotAllowed,
	1500: http.StatusInternalServerError, 1501: http.StatusServiceUnavailable,
	2000: http.StatusUnauthorized, 2001: http.StatusUnauthorized, 2002: http.StatusUnauthorized,
	2003: http.StatusForbidden, 2004: http.StatusForbidden, 2005: http.StatusBadRequest,
	2006: http.StatusForbidden, 2007: http.StatusUnauthorized, 2008: http.StatusForbidden,
}

// OK 通过统一出口编码成功响应。
func OK[T any](c *gin.Context, data T) {
	write(c, http.StatusOK, Body[T]{Code: 0, Msg: "ok", Data: data})
}

// Fail 识别包装后的业务错误，未知错误不透出内部细节。
func Fail(c *gin.Context, err error) {
	business, ok := errors.AsType[*BizError](err)
	if !ok || business == nil {
		business = ErrInternal
	}
	status, known := statuses[business.Code]
	if !known {
		business, status = ErrInternal, http.StatusInternalServerError
	}
	c.Abort()
	write(c, status, Body[any]{Code: business.Code, Msg: business.Msg, Data: nil})
}

func write(c *gin.Context, status int, body any) {
	switch b := body.(type) {
	case Body[any]:
		c.Set("responseCode", b.Code)
		c.Set("responseMessage", b.Msg)
	default:
		c.Set("responseCode", 0)
		c.Set("responseMessage", "ok")
	}
	data, err := json.Marshal(body)
	if err != nil {
		status = http.StatusInternalServerError
		data = []byte(`{"code":1500,"msg":"服务内部错误","data":null}`)
		c.Abort()
	}
	c.Data(status, "application/json; charset=utf-8", data)
}

// PageQuery 对公共分页参数进行边界校验。
type PageQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	OrderBy  string `form:"orderBy"`
}

// Normalize 应用默认值，拒绝负数与超出上限的请求。
func (q *PageQuery) Normalize() error {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.Page < 1 || q.PageSize < 1 || q.PageSize > 100 || q.Page > 1_000_000_000 {
		return ErrInvalidParam
	}
	return nil
}

// Sort 将客户端排序项映射为服务器可信列名和方向。
func (q PageQuery) Sort(columns map[string]string) (column string, descending bool, err error) {
	if q.OrderBy == "" {
		return "", false, nil
	}
	parts := strings.Fields(q.OrderBy)
	if len(parts) < 1 || len(parts) > 2 {
		return "", false, ErrInvalidParam
	}
	column, ok := columns[parts[0]]
	if !ok {
		return "", false, ErrInvalidParam
	}
	if len(parts) == 2 {
		switch strings.ToLower(parts[1]) {
		case "asc":
		case "desc":
			descending = true
		default:
			return "", false, ErrInvalidParam
		}
	}
	return column, descending, nil
}
