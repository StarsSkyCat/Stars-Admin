package config

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 只绑定参数并调用参数业务。
type Handler struct{ service *Service }

// NewHandler 创建参数 HTTP 入口。
func NewHandler(service *Service) *Handler { return &Handler{service} }

// List 分页查询参数。
// @Summary 分页查询参数
// @Tags 参数管理
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最多 100"
// @Param orderBy query string false "白名单排序字段及方向"
// @Success 200 {object} response.Body[response.Page[View]]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/configs [get]
func (h *Handler) List(c *gin.Context) {
	var q response.PageQuery
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

// Get 查询参数详情。
// @Summary 查询参数详情
// @Tags 参数管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/configs/{id} [get]
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

// Create 创建参数。
// @Summary 创建参数
// @Tags 参数管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/configs [post]
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

// Update 编辑参数资料。
// @Summary 编辑参数资料
// @Tags 参数管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/configs/{id} [put]
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

// Remove 批量删除参数。
// @Summary 批量删除参数
// @Tags 参数管理
// @Security BearerAuth
// @Produce json
// @Param ids path string true "逗号分隔的 ID，最多 100 个"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/configs/{ids} [delete]
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

// RegisterRoutes 将每个动作与具体权限标识绑定。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, required gin.HandlerFunc, permit func(string) gin.HandlerFunc) {
	rg.Use(required)
	rg.GET("/key/:key", permit("sys:config:get"), h.ByKey)
	rg.POST("/cache/refresh", permit("sys:config:refresh"), h.Refresh)
	rg.GET("", permit("sys:config:list"), h.List)
	rg.GET("/:id", permit("sys:config:get"), h.Get)
	rg.POST("", permit("sys:config:add"), h.Create)
	rg.PUT("/:id", permit("sys:config:edit"), h.Update)
	rg.DELETE("/:ids", permit("sys:config:remove"), h.Remove)
}

// ByKey 按键读取参数。
// @Summary 按键读取参数
// @Tags 参数管理
// @Security BearerAuth
// @Produce json
// @Param key path string true "参数键"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,503 {object} response.Body[any]
// @Router /sys/configs/key/{key} [get]
func (h *Handler) ByKey(c *gin.Context) {
	out, err := h.service.ByKey(c.Request.Context(), c.Param("key"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Refresh 清理普通参数缓存。
// @Summary 清理参数缓存
// @Tags 参数管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Status]
// @Failure 401,403,503 {object} response.Body[any]
// @Router /sys/configs/cache/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	if err := h.service.Refresh(c.Request.Context()); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}
