package table

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 只绑定参数并调用生成配置业务。
type Handler struct{ service *Service }

// NewHandler 创建生成配置 HTTP 入口。
func NewHandler(service *Service) *Handler { return &Handler{service} }

// List 分页查询生成配置。
// @Summary 分页查询生成配置
// @Tags 生成配置管理
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最多 100"
// @Param orderBy query string false "白名单排序字段及方向"
// @Success 200 {object} response.Body[response.Page[View]]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /gen/tables [get]
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

// Get 查询生成配置详情。
// @Summary 查询生成配置详情
// @Tags 生成配置管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /gen/tables/{id} [get]
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

// Update 编辑生成配置资料。
// @Summary 编辑生成配置资料
// @Tags 生成配置管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "字符串 ID"
// @Param body body Input true "业务参数"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /gen/tables/{id} [put]
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

// Remove 批量删除生成配置。
// @Summary 批量删除生成配置
// @Tags 生成配置管理
// @Security BearerAuth
// @Produce json
// @Param ids path string true "逗号分隔的 ID，最多 100 个"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /gen/tables/{ids} [delete]
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

// RegisterRoutes 仅装配已认证、逐动作授权的管理入口。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, g httpx.Guards) {
	g = g.Normalize()
	rg.Use(g.Required, g.Operations)
	rg.GET("", g.Permit("gen:table:list"), h.List)
	rg.GET("/db", g.Permit("gen:table:list"), h.DB)
	rg.POST("/import", g.Permit("gen:table:import"), h.Import)
	rg.GET("/:id", g.Permit("gen:table:get"), h.Get)
	rg.PUT("/:id", g.Permit("gen:table:edit"), h.Update)
	rg.DELETE("/:ids", g.Permit("gen:table:remove"), h.Remove)
	rg.POST("/:id/sync", g.Permit("gen:table:edit"), h.Sync)
	rg.GET("/:id/preview", g.Permit("gen:table:preview"), h.Preview)
	rg.POST("/:id/generate", g.Permit("gen:table:gen"), g.Duplicate, h.Generate)
}

// DB 返回可导入的业务表。
// @Summary 查询可导入的数据库表
// @Tags 代码生成
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[[]DBTable]
// @Failure 401,403,503 {object} response.Body[any]
// @Router /gen/tables/db [get]
func (h *Handler) DB(c *gin.Context) {
	out, err := h.service.DBTables(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Import 导入元数据，不导入业务数据。
// @Summary 导入数据库表结构
// @Tags 代码生成
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body ImportInput true "表名列表"
// @Success 200 {object} response.Body[[]View]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /gen/tables/import [post]
func (h *Handler) Import(c *gin.Context) {
	var in ImportInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Import(c.Request.Context(), httpx.Actor(c), in.Tables)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Sync 更新结构并标记配置须重新确认。
// @Summary 同步数据库表结构
// @Tags 代码生成
// @Security BearerAuth
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,404,503 {object} response.Body[any]
// @Router /gen/tables/{id}/sync [post]
func (h *Handler) Sync(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Sync(c.Request.Context(), httpx.Actor(c), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Preview 返回固定模板的完整待审文本。
// @Summary 预览生成代码
// @Tags 代码生成
// @Security BearerAuth
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} response.Body[[]GeneratedFile]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /gen/tables/{id}/preview [get]
func (h *Handler) Preview(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Preview(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Generate 返回 zip；显式开启的非生产 local 模式返回 JSON 文件清单。
// @Summary 生成代码包
// @Tags 代码生成
// @Security BearerAuth
// @Produce application/zip,json
// @Param id path string true "配置ID"
// @Success 200 {file} binary
// @Description 默认 zip 流；显式 local 模式返回统一 JSON 文件清单。
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /gen/tables/{id}/generate [post]
func (h *Handler) Generate(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Generate(c.Request.Context(), httpx.Actor(c), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if out.Local != nil {
		response.OK(c, out.Local)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="generated.zip"`)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(200, "application/zip", out.ZIP)
}
