package audit

import (
	"net/http"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 绑定审计请求，不访问数据库或 Redis。
type Handler struct{ service *Service }

// NewHandler 创建审计入口。
func NewHandler(s *Service) *Handler { return &Handler{s} }

// ListOper 处理审计请求。
// @Summary Oper 日志List
// @Tags 日志审计
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Page[OperView]]
// @Failure 400,401,403,503 {object} response.Body[any]
// @Router /sys/audit/oper-logs [get]
func (h *Handler) ListOper(c *gin.Context) {
	var q Query
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	out, err := h.service.Operations(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// CleanOper 处理审计请求。
// @Summary Oper 日志Clean
// @Tags 日志审计
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,503 {object} response.Body[any]
// @Router /sys/audit/oper-logs/clean [delete]
func (h *Handler) CleanOper(c *gin.Context) {
	if err := h.service.Clean(c.Request.Context(), true); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// ExportOper 处理审计请求。
// @Summary Oper 日志Export
// @Tags 日志审计
// @Security BearerAuth
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {file} binary
// @Failure 400,401,403,503 {object} response.Body[any]
// @Router /sys/audit/oper-logs/export [post]
func (h *Handler) ExportOper(c *gin.Context) {
	var q Query
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	out, err := h.service.Export(c.Request.Context(), q, true)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=logs.xlsx")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", out)
}

// ListLogin 处理审计请求。
// @Summary Login 日志List
// @Tags 日志审计
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Page[LoginView]]
// @Failure 400,401,403,503 {object} response.Body[any]
// @Router /sys/audit/login-logs [get]
func (h *Handler) ListLogin(c *gin.Context) {
	var q Query
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	out, err := h.service.Logins(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// CleanLogin 处理审计请求。
// @Summary Login 日志Clean
// @Tags 日志审计
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,503 {object} response.Body[any]
// @Router /sys/audit/login-logs/clean [delete]
func (h *Handler) CleanLogin(c *gin.Context) {
	if err := h.service.Clean(c.Request.Context(), false); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// ExportLogin 处理审计请求。
// @Summary Login 日志Export
// @Tags 日志审计
// @Security BearerAuth
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {file} binary
// @Failure 400,401,403,503 {object} response.Body[any]
// @Router /sys/audit/login-logs/export [post]
func (h *Handler) ExportLogin(c *gin.Context) {
	var q Query
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	out, err := h.service.Export(c.Request.Context(), q, false)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=logs.xlsx")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", out)
}

// Online 查询有效在线会话。
// @Summary 查询在线会话
// @Tags 日志审计
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Page[OnlineView]]
// @Failure 400,401,403,503 {object} response.Body[any]
// @Router /sys/audit/online [get]
func (h *Handler) Online(c *gin.Context) {
	var q response.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	out, err := h.service.Online(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// ForceLogout 撤销目标用户全部会话。
// @Summary 强制用户全部会话下线
// @Tags 日志审计
// @Security BearerAuth
// @Produce json
// @Param userId path string true "用户 ID"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,503 {object} response.Body[any]
// @Router /sys/audit/online/{userId} [delete]
func (h *Handler) ForceLogout(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("userId"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.ForceLogout(c.Request.Context(), httpx.Actor(c), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// RegisterRoutes 危险操作使用独立权限，默认仅 admin。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, required gin.HandlerFunc, permit func(string) gin.HandlerFunc) {
	rg.Use(required)
	rg.GET("/online", permit("sys:audit:onlineList"), h.Online)
	rg.DELETE("/online/:userId", permit("sys:audit:forceLogout"), h.ForceLogout)
	rg.GET("/oper-logs", permit("sys:audit:operList"), h.ListOper)
	rg.DELETE("/oper-logs/clean", permit("sys:audit:operClean"), h.CleanOper)
	rg.POST("/oper-logs/export", permit("sys:audit:operExport"), h.ExportOper)
	rg.GET("/login-logs", permit("sys:audit:loginList"), h.ListLogin)
	rg.DELETE("/login-logs/clean", permit("sys:audit:loginClean"), h.CleanLogin)
	rg.POST("/login-logs/export", permit("sys:audit:loginExport"), h.ExportLogin)
}
