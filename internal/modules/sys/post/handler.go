package post

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 只绑定参数并调用岗位业务。
type Handler struct{ service *Service }

// NewHandler 创建岗位 HTTP 入口。
func NewHandler(service *Service) *Handler { return &Handler{service} }

// List 分页查询岗位。
// @Summary 分页查询岗位
// @Tags 岗位管理
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最多 100"
// @Param orderBy query string false "白名单排序字段及方向"
// @Success 200 {object} response.Body[response.Page[View]]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/posts [get]
func (h *Handler) List(c *gin.Context) {
	var q response.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	out, err := h.service.List(c.Request.Context(), q, false)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Get 查询岗位详情。
// @Summary 查询岗位详情
// @Tags 岗位管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/posts/{id} [get]
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

// Create 创建岗位。
// @Summary 创建岗位
// @Tags 岗位管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/posts [post]
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

// Update 编辑岗位资料。
// @Summary 编辑岗位资料
// @Tags 岗位管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/posts/{id} [put]
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

// Remove 批量删除岗位。
// @Summary 批量删除岗位
// @Tags 岗位管理
// @Security BearerAuth
// @Produce json
// @Param ids path string true "逗号分隔的 ID，最多 100 个"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/posts/{ids} [delete]
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

// Options 查询岗位选项。
// @Summary 查询岗位选项
// @Tags 岗位管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[[]View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/posts/options [get]
func (h *Handler) Options(c *gin.Context) {
	out, err := h.service.List(c.Request.Context(), response.PageQuery{}, true)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out.List)
}

// RegisterRoutes 将每个动作与具体权限标识绑定。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, required gin.HandlerFunc, permit func(string) gin.HandlerFunc) {
	rg.Use(required)
	rg.GET("", permit("sys:post:list"), h.List)
	rg.GET("/:id", permit("sys:post:get"), h.Get)
	rg.POST("", permit("sys:post:add"), h.Create)
	rg.PUT("/:id", permit("sys:post:edit"), h.Update)
	rg.DELETE("/:ids", permit("sys:post:remove"), h.Remove)
	rg.GET("/options", permit("sys:post:list"), h.Options)

}
