package audit

import (
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/session"
)

// Query 只允许用户名和状态白名单查询。
type Query struct {
	response.PageQuery
	Username string `form:"username"`
	Status   int    `form:"status"`
	export   bool
}

// LoginView 是登录日志公开字段。
type LoginView struct {
	ID       model.ID  `json:"id"`
	Username string    `json:"username"`
	IP       string    `json:"ip"`
	Status   int       `json:"status"`
	Msg      string    `json:"msg"`
	LoginAt  time.Time `json:"loginAt"`
}

// OperView 仅包含已脱敏操作元信息。
type OperView struct {
	ID       model.ID  `json:"id"`
	UserID   model.ID  `json:"userId"`
	Username string    `json:"username"`
	Module   string    `json:"module"`
	Action   string    `json:"action"`
	TraceID  string    `json:"traceId"`
	Method   string    `json:"method"`
	URL      string    `json:"url"`
	IP       string    `json:"ip"`
	Response string    `json:"response"`
	Status   int       `json:"status"`
	CostMS   int       `json:"costMs"`
	OperAt   time.Time `json:"operAt"`
}

// OnlineView 是在线会话的公开摘要。
type OnlineView session.Online
