package httpx

import (
	"context"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Guards 在模块装配时注入认证、逐动作授权和横切能力，不依赖 sys 实现。
type Guards struct {
	Required   gin.HandlerFunc
	Permit     func(string) gin.HandlerFunc
	Operations gin.HandlerFunc
	Duplicate  gin.HandlerFunc
}

// Normalize 对缺失的身份或授权能力默认拒绝。
func (g Guards) Normalize() Guards {
	deny := func(c *gin.Context) { response.Fail(c, response.ErrPermission) }
	pass := func(c *gin.Context) { c.Next() }
	if g.Required == nil {
		g.Required = deny
	}
	if g.Permit == nil {
		g.Permit = func(string) gin.HandlerFunc { return deny }
	}
	if g.Operations == nil {
		g.Operations = pass
	}
	if g.Duplicate == nil {
		g.Duplicate = deny
	}
	return g
}

// Deadline 为模块请求建立超时，异步任务须另建受生命周期管理的 context。
func Deadline(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
