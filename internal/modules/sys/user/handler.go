package user

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 只绑定参数并调用用户业务。
type Handler struct{ service *Service }

// NewHandler 创建用户 HTTP 入口。
func NewHandler(service *Service) *Handler { return &Handler{service} }

// List 分页查询用户。
// @Summary 分页查询用户
// @Tags 用户管理
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最多 100"
// @Param orderBy query string false "白名单排序字段及方向"
// @Success 200 {object} response.Body[response.Page[View]]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/users [get]
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

// Get 查询用户详情。
// @Summary 查询用户详情
// @Tags 用户管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/users/{id} [get]
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

// Create 创建用户。
// @Summary 创建用户
// @Tags 用户管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body CreateInput true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/users [post]
func (h *Handler) Create(c *gin.Context) {
	var in CreateInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Create(c.Request.Context(), httpx.Actor(c), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Update 编辑用户资料。
// @Summary 编辑用户资料
// @Description 普通用户修改本人部门时，新数据范围必须包含于原范围。
// @Tags 用户管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/users/{id} [put]
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

// Remove 批量删除用户。
// @Summary 批量删除用户
// @Tags 用户管理
// @Security BearerAuth
// @Produce json
// @Param ids path string true "逗号分隔的 ID，最多 100 个"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/users/{ids} [delete]
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
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, required gin.HandlerFunc, permit func(string) gin.HandlerFunc, duplicate ...gin.HandlerFunc) {
	rg.Use(required)
	lock := func(c *gin.Context) { c.Next() }
	if len(duplicate) > 0 {
		lock = duplicate[0]
	}
	rg.POST("/import", permit("sys:user:import"), lock, h.Import)
	rg.POST("/export", permit("sys:user:export"), h.Export)
	rg.GET("", permit("sys:user:list"), h.List)
	rg.GET("/:id", permit("sys:user:get"), h.Get)
	rg.POST("", permit("sys:user:add"), h.Create)
	rg.PUT("/:id", permit("sys:user:edit"), h.Update)
	rg.DELETE("/:ids", permit("sys:user:remove"), h.Remove)

	rg.PUT("/:id/status", permit("sys:user:edit"), h.Status)
	rg.PUT("/:id/password", permit("sys:user:resetPwd"), lock, h.Password)
	rg.PUT("/:id/roles", permit("sys:user:grant"), h.Roles)
	rg.PUT("/:id/posts", permit("sys:user:grant"), h.Posts)

}

// Status 修改用户状态并撤销会话。
// @Summary 修改用户状态并撤销会话
// @Tags 用户管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body StatusInput true "业务参数"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/users/{id}/status [put]
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

// Password 重置用户密码并强制首登改密。
// @Summary 重置用户密码并强制首登改密
// @Tags 用户管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body PasswordInput true "业务参数"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/users/{id}/password [put]
func (h *Handler) Password(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in PasswordInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), httpx.Actor(c), id, in.Password); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// Roles 替换用户角色授权并撤销会话。
// @Summary 替换用户角色授权并撤销会话
// @Tags 用户管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body RolesInput true "业务参数"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/users/{id}/roles [put]
func (h *Handler) Roles(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in RolesInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.GrantRoles(c.Request.Context(), httpx.Actor(c), id, in.RoleIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// Posts 替换岗位绑定。
// @Summary 替换用户岗位
// @Tags 用户管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "用户 ID"
// @Param body body PostsInput true "岗位 ID 列表"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/users/{id}/posts [put]
func (h *Handler) Posts(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in PostsInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.GrantPosts(c.Request.Context(), httpx.Actor(c), id, in.PostIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}
