package middleware

import (
	"encoding/json/v2"
	"strings"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/gin-gonic/gin"
)

// AuditEvent 仅包含复制后的白名单字段，不含请求上下文或原始内容。
type AuditEvent struct {
	UserID   model.ID
	Username string
	Module   string
	Action   string
	TraceID  string
	Method   string
	URL      string
	IP       string
	Response string
	Status   int
	CostMS   int
	OperAt   time.Time
}

// AuditSink 接收尽力投递，禁止阻塞请求。
type AuditSink interface{ Submit(AuditEvent) }

// OperLog 采集写操作元信息，认证端点、文件及 Excel 流均不读取原文。
func OperLog(sink AuditSink) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			return
		}
		route := c.FullPath()
		if route == "" || strings.HasPrefix(route, "/api/sys/auth/") {
			return
		}
		p := security.Actor(c.Request.Context())
		code, _ := c.Get("responseCode")
		msg, _ := c.Get("responseMessage")
		summary, _ := json.Marshal(struct {
			Code    any `json:"code"`
			Message any `json:"msg"`
		}{code, msg})
		status := 1
		if c.Writer.Status() >= 400 {
			status = 2
		}
		sink.Submit(AuditEvent{UserID: p.ID, Username: p.Username, Module: "sys", Action: route, TraceID: c.GetString(requestIDKey), Method: c.Request.Method, URL: route, IP: c.ClientIP(), Response: string(summary), Status: status, CostMS: int(min(time.Since(start).Milliseconds(), int64(2147483647))), OperAt: start.UTC()})
	}
}
