package menu

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 只绑定参数并调用菜单业务。
type Handler struct{ service *Service }

// NewHandler 创建菜单 HTTP 入口。
func NewHandler(service *Service) *Handler { return &Handler{service} }

// List 分页查询菜单。
// @Summary 分页查询菜单
// @Tags 菜单管理
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最多 100"
// @Param orderBy query string false "白名单排序字段及方向"
// @Success 200 {object} response.Body[response.Page[View]]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/menus [get]
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

// Get 查询菜单详情。
// @Summary 查询菜单详情
// @Tags 菜单管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/menus/{id} [get]
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

// Create 创建菜单。
// @Summary 创建菜单
// @Tags 菜单管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/menus [post]
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

// Update 编辑菜单资料。
// @Summary 编辑菜单资料
// @Tags 菜单管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/menus/{id} [put]
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

// Remove 批量删除菜单。
// @Summary 批量删除菜单
// @Tags 菜单管理
// @Security BearerAuth
// @Produce json
// @Param ids path string true "逗号分隔的 ID，最多 100 个"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/menus/{ids} [delete]
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

// Tree 查询菜单树。
// @Summary 查询菜单树
// @Tags 菜单管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[[]View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/menus/tree [get]
// @Router /sys/menus/options [get]
func (h *Handler) Tree(c *gin.Context) {
	out, err := h.service.Tree(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// RegisterRoutes 将每个动作与具体权限标识绑定。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, required gin.HandlerFunc, permit func(string) gin.HandlerFunc) {
	rg.Use(required)
	rg.GET("", permit("sys:menu:list"), h.List)
	rg.GET("/:id", permit("sys:menu:get"), h.Get)
	rg.POST("", permit("sys:menu:add"), h.Create)
	rg.PUT("/:id", permit("sys:menu:edit"), h.Update)
	rg.DELETE("/:ids", permit("sys:menu:remove"), h.Remove)
	rg.GET("/tree", permit("sys:menu:list"), h.Tree)
	rg.GET("/options", permit("sys:menu:list"), h.Tree)

}
