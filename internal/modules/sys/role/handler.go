package role

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 只绑定参数并调用角色业务。
type Handler struct{ service *Service }

// NewHandler 创建角色 HTTP 入口。
func NewHandler(service *Service) *Handler { return &Handler{service} }

// List 分页查询角色。
// @Summary 分页查询角色
// @Tags 角色管理
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最多 100"
// @Param orderBy query string false "白名单排序字段及方向"
// @Success 200 {object} response.Body[response.Page[View]]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/roles [get]
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

// Get 查询角色详情。
// @Summary 查询角色详情
// @Tags 角色管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/roles/{id} [get]
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

// Create 创建角色。
// @Summary 创建角色
// @Tags 角色管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/roles [post]
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

// Update 编辑角色资料。
// @Summary 编辑角色资料
// @Tags 角色管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/roles/{id} [put]
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

// Remove 批量删除角色。
// @Summary 批量删除角色
// @Tags 角色管理
// @Security BearerAuth
// @Produce json
// @Param ids path string true "逗号分隔的 ID，最多 100 个"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/roles/{ids} [delete]
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
	rg.GET("", permit("sys:role:list"), h.List)
	rg.GET("/:id", permit("sys:role:get"), h.Get)
	rg.POST("", permit("sys:role:add"), h.Create)
	rg.PUT("/:id", permit("sys:role:edit"), h.Update)
	rg.DELETE("/:ids", permit("sys:role:remove"), h.Remove)

	rg.PUT("/:id/status", permit("sys:role:edit"), h.Status)
	rg.PUT("/:id/menus", permit("sys:role:grant"), h.Menus)
	rg.PUT("/:id/data-scope", permit("sys:role:grant"), h.Scope)
}

// Status 修改角色状态并撤销成员会话。
// @Summary 修改角色状态并撤销成员会话
// @Tags 角色管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body StatusInput true "业务参数"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/roles/{id}/status [put]
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

// Menus 替换角色菜单授权。
// @Summary 替换角色菜单授权
// @Tags 角色管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body MenuInput true "业务参数"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/roles/{id}/menus [put]
func (h *Handler) Menus(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in MenuInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.GrantMenus(c.Request.Context(), httpx.Actor(c), id, in.MenuIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// Scope 配置角色数据范围。
// @Summary 配置角色数据范围
// @Tags 角色管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body ScopeInput true "业务参数"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/roles/{id}/data-scope [put]
func (h *Handler) Scope(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in ScopeInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.GrantScope(c.Request.Context(), httpx.Actor(c), id, in); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}
