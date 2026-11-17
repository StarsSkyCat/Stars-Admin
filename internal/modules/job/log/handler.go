package log

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 仅绑定任务日志查询参数。
type Handler struct{ service *Service }

// NewHandler 创建日志 HTTP 入口。
func NewHandler(s *Service) *Handler { return &Handler{s} }

// List 查询执行日志。
// @Summary 查询任务执行日志
// @Tags 任务日志
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param taskId query string false "任务ID"
// @Success 200 {object} response.Body[response.Page[View]]
// @Failure 400,401,403,503 {object} response.Body[any]
// @Router /job/logs [get]
func (h *Handler) List(c *gin.Context) {
	var q Query
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	out, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Clean 清空已落库日志，后续任务可能生成新日志。
// @Summary 清空任务日志
// @Tags 任务日志
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Status]
// @Failure 401,403,503 {object} response.Body[any]
// @Router /job/logs/clean [delete]
func (h *Handler) Clean(c *gin.Context) {
	if err := h.service.Clean(c.Request.Context()); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// RegisterRoutes 日志清空绑定独立权限。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, g httpx.Guards) {
	g = g.Normalize()
	rg.Use(g.Required, g.Operations)
	rg.GET("", g.Permit("job:log:list"), h.List)
	rg.DELETE("/clean", g.Permit("job:log:remove"), h.Clean)
}
