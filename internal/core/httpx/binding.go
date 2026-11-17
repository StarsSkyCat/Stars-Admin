// Package httpx 提供有界请求解码及可信上下文提取。
package httpx

import (
	"encoding/json/v2"
	"io"
	"strconv"
	"strings"
	"time"

	"errors"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm"
)

// Meta 是实体公开元信息，与 GORM 模型分离。
type Meta struct {
	ID        model.ID  `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Bind 拒绝未知字段和过大请求，防止通用编辑绕过专用授权入口。
func Bind(c *gin.Context, dst any) error {
	data, err := io.ReadAll(io.LimitReader(c.Request.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return response.ErrInvalidParam
	}
	if err := json.Unmarshal(data, dst, json.RejectUnknownMembers(true)); err != nil {
		return response.ErrInvalidParam
	}
	if err := binding.Validator.ValidateStruct(dst); err != nil {
		return response.ErrInvalidParam
	}
	return nil
}

// ParseID 只接受正整数的标准十进制表示。
func ParseID(text string) (model.ID, error) {
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != text {
		return 0, response.ErrInvalidParam
	}
	return model.ID(n), nil
}

// IDs 拒绝重复、空项及超过一百项的批量请求。
func IDs(text string) ([]model.ID, error) {
	values := strings.Split(text, ",")
	if len(values) > 100 {
		return nil, response.ErrInvalidParam
	}
	ids := make([]model.ID, 0, len(values))
	seen := map[model.ID]bool{}
	for _, v := range values {
		id, err := ParseID(v)
		if err != nil || seen[id] {
			return nil, response.ErrInvalidParam
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}

// UniqueIDs 校验 JSON ID 列表，空列表表示显式解绑。
func UniqueIDs(ids []model.ID) error {
	if len(ids) > 100 {
		return response.ErrInvalidParam
	}
	seen := map[model.ID]bool{}
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return response.ErrInvalidParam
		}
		seen[id] = true
	}
	return nil
}

// Actor 只能由认证中间件写入，业务层不接受 HTTP DTO 中的操作者。
func Actor(c *gin.Context) security.Principal {
	value, _ := c.Get("principal")
	p, _ := value.(security.Principal)
	return p
}

// DBError 映射可预期约束，不向 HTTP 暴露数据库内部信息。
func DBError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.ErrNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) || errors.Is(err, gorm.ErrForeignKeyViolated) {
		return response.ErrConflict
	}
	return err
}
