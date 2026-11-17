package dict

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 绑定字典管理参数。
type Handler struct{ service *Service }

// NewHandler 创建字典 HTTP 入口。
func NewHandler(s *Service) *Handler { return &Handler{s} }

// ListType 处理字典类型请求。
// @Summary 字典类型List
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Page[TypeView]]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/dict/types [get]
func (h *Handler) ListType(c *gin.Context) {
	var q response.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	out, err := h.service.ListTypes(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// GetType 处理字典类型请求。
// @Summary 字典类型Get
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Success 200 {object} response.Body[TypeView]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/dict/types/{id} [get]
func (h *Handler) GetType(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.GetType(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// CreateType 处理字典类型请求。
// @Summary 字典类型Create
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Accept json
// @Param body body TypeInput true "业务参数"
// @Success 200 {object} response.Body[TypeView]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/dict/types [post]
func (h *Handler) CreateType(c *gin.Context) {
	var in TypeInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.SaveType(c.Request.Context(), httpx.Actor(c), 0, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// UpdateType 处理字典类型请求。
// @Summary 字典类型Update
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Accept json
// @Param body body TypeInput true "业务参数"
// @Success 200 {object} response.Body[TypeView]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/dict/types/{id} [put]
func (h *Handler) UpdateType(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in TypeInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.SaveType(c.Request.Context(), httpx.Actor(c), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// RemoveType 处理字典类型请求。
// @Summary 字典类型Remove
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Param ids path string true "逗号分隔 ID，最多 100"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/dict/types/{ids} [delete]
func (h *Handler) RemoveType(c *gin.Context) {
	ids, err := httpx.IDs(c.Param("ids"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	err = h.service.Remove(c.Request.Context(), httpx.Actor(c), ids, false)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// ListData 处理字典数据请求。
// @Summary 字典数据List
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Page[DataView]]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/dict/data [get]
func (h *Handler) ListData(c *gin.Context) {
	var q response.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	out, err := h.service.ListData(c.Request.Context(), q, c.Query("dictType"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// GetData 处理字典数据请求。
// @Summary 字典数据Get
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Success 200 {object} response.Body[DataView]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/dict/data/{id} [get]
func (h *Handler) GetData(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.GetData(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// CreateData 处理字典数据请求。
// @Summary 字典数据Create
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Accept json
// @Param body body DataInput true "业务参数"
// @Success 200 {object} response.Body[DataView]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/dict/data [post]
func (h *Handler) CreateData(c *gin.Context) {
	var in DataInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.SaveData(c.Request.Context(), httpx.Actor(c), 0, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// UpdateData 处理字典数据请求。
// @Summary 字典数据Update
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "字符串 ID"
// @Accept json
// @Param body body DataInput true "业务参数"
// @Success 200 {object} response.Body[DataView]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/dict/data/{id} [put]
func (h *Handler) UpdateData(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in DataInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.SaveData(c.Request.Context(), httpx.Actor(c), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// RemoveData 处理字典数据请求。
// @Summary 字典数据Remove
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Param ids path string true "逗号分隔 ID，最多 100"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,409,503 {object} response.Body[any]
// @Router /sys/dict/data/{ids} [delete]
func (h *Handler) RemoveData(c *gin.Context) {
	ids, err := httpx.IDs(c.Param("ids"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	err = h.service.Remove(c.Request.Context(), httpx.Actor(c), ids, true)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// ByType 读取启用字典项，登录后可用。
// @Summary 按类型读取启用字典项
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Param type path string true "类型标识"
// @Success 200 {object} response.Body[[]DataView]
// @Failure 400,401,404,503 {object} response.Body[any]
// @Router /sys/dict/data/type/{type} [get]
func (h *Handler) ByType(c *gin.Context) {
	out, err := h.service.ByType(c.Request.Context(), c.Param("type"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Refresh 失效字典缓存。
// @Summary 清理字典缓存
// @Tags 字典管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Status]
// @Failure 401,403,503 {object} response.Body[any]
// @Router /sys/dict/cache/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	if err := h.service.Refresh(c.Request.Context()); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// RegisterRoutes 将字典动作绑定至独立权限。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, required gin.HandlerFunc, permit func(string) gin.HandlerFunc) {
	rg.Use(required)
	rg.GET("/data/type/:type", h.ByType)
	rg.POST("/cache/refresh", permit("sys:dict:refresh"), h.Refresh)
	rg.GET("/types", permit("sys:dict:list"), h.ListType)
	rg.GET("/types/:id", permit("sys:dict:get"), h.GetType)
	rg.POST("/types", permit("sys:dict:add"), h.CreateType)
	rg.PUT("/types/:id", permit("sys:dict:edit"), h.UpdateType)
	rg.DELETE("/types/:ids", permit("sys:dict:remove"), h.RemoveType)
	rg.GET("/data", permit("sys:dict:list"), h.ListData)
	rg.GET("/data/:id", permit("sys:dict:get"), h.GetData)
	rg.POST("/data", permit("sys:dict:add"), h.CreateData)
	rg.PUT("/data/:id", permit("sys:dict:edit"), h.UpdateData)
	rg.DELETE("/data/:ids", permit("sys:dict:remove"), h.RemoveData)
}
