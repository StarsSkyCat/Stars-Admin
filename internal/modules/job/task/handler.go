package task

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 只绑定参数并调用任务业务。
type Handler struct{ service *Service }

// NewHandler 创建任务 HTTP 入口。
func NewHandler(service *Service) *Handler { return &Handler{service} }

// List 分页查询任务。
// @Summary 分页查询任务
// @Tags 任务管理
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最多 100"
// @Param orderBy query string false "白名单排序字段及方向"
// @Success 200 {object} response.Body[response.Page[View]]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /job/tasks [get]
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

// Get 查询任务详情。
// @Summary 查询任务详情
// @Tags 任务管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /job/tasks/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Create 创建任务。
// @Summary 创建任务
// @Tags 任务管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /job/tasks [post]
func (h *Handler) Create(c *gin.Context) {
	var in Input
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Save(c.Request.Context(), httpx.Actor(c), 0, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Update 编辑任务资料。
// @Summary 编辑任务资料
// @Tags 任务管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /job/tasks/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in Input
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Save(c.Request.Context(), httpx.Actor(c), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Remove 批量删除任务。
// @Summary 批量删除任务
// @Tags 任务管理
// @Security BearerAuth
// @Produce json
// @Param ids path string true "逗号分隔的 ID，最多 100 个"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /job/tasks/{ids} [delete]
func (h *Handler) Remove(c *gin.Context) {
	ids, err := httpx.IDs(c.Param("ids"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.Remove(c.Request.Context(), httpx.Actor(c), ids); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// RegisterRoutes 手动执行使用独立权限和防重复提交。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, g httpx.Guards) {
	g = g.Normalize()
	rg.Use(g.Required, g.Operations)
	rg.GET("", g.Permit("job:task:list"), h.List)
	rg.GET("/cron/next", g.Permit("job:task:list"), h.Next)
	rg.GET("/:id", g.Permit("job:task:get"), h.Get)
	rg.POST("", g.Permit("job:task:add"), h.Create)
	rg.PUT("/:id", g.Permit("job:task:edit"), h.Update)
	rg.DELETE("/:ids", g.Permit("job:task:remove"), h.Remove)
	rg.PUT("/:id/status", g.Permit("job:task:edit"), h.Status)
	rg.POST("/:id/run", g.Permit("job:task:run"), g.Duplicate, h.Run)
}

// Status 启用或暂停配置。
// @Summary 启用或暂停任务
// @Tags 任务管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param body body StatusInput true "状态"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,503 {object} response.Body[any]
// @Router /job/tasks/{id}/status [put]
func (h *Handler) Status(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in StatusInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.Status(c.Request.Context(), httpx.Actor(c), id, in.Status); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// Run 手动执行返回投递或跳过状态。
// @Summary 手动执行任务
// @Tags 任务管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Body[RunResult]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /job/tasks/{id}/run [post]
func (h *Handler) Run(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Run(c.Request.Context(), httpx.Actor(c), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Next 使用同一解析器计算未来五次 UTC 触发。
// @Summary 预览Cron未来五次触发
// @Tags 任务管理
// @Security BearerAuth
// @Produce json
// @Param expr query string true "六位Cron"
// @Success 200 {object} response.Body[[]string]
// @Failure 400,401,403,503 {object} response.Body[any]
// @Router /job/tasks/cron/next [get]
func (h *Handler) Next(c *gin.Context) {
	out, err := h.service.Next(c.Query("expr"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}
