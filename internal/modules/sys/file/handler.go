package file

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Handler 只执行文件请求绑定及流响应。
type Handler struct {
	service *Service
	maxSize int64
}

// NewHandler 创建受保护文件入口。
func NewHandler(s *Service, maxSize int64) *Handler { return &Handler{s, maxSize} }

// Upload 上传并校验文件。
// @Summary 上传文件
// @Tags 文件管理
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "文件，默认最多 10MB"
// @Success 200 {object} response.Body[View]
// @Failure 400,401,403,409,503 {object} response.Body[any]
// @Router /sys/files/upload [post]
func (h *Handler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxSize+1024*1024)
	if err := c.Request.ParseMultipartForm(1024 * 1024); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	if c.Request.MultipartForm != nil {
		defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	}
	header, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	body, err := header.Open()
	if err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	defer func() { _ = body.Close() }()
	out, err := h.service.Upload(c.Request.Context(), httpx.Actor(c), header.Filename, body)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Download 返回已授权的附件流。
// @Summary 下载文件
// @Tags 文件管理
// @Security BearerAuth
// @Produce application/octet-stream
// @Param id path string true "文件 ID"
// @Success 200 {file} binary
// @Failure 400,401,403,404,503 {object} response.Body[any]
// @Router /sys/files/{id}/download [get]
func (h *Handler) Download(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Download(c.Request.Context(), httpx.Actor(c), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	defer func() { _ = out.Body.Close() }()
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": out.OriginalName}))
	c.Header("Content-Length", strconv.FormatInt(out.Size, 10))
	c.DataFromReader(http.StatusOK, out.Size, "application/octet-stream", out.Body, nil)
}

// Remove 删除文件并持久记录物理删除补偿。
// @Summary 删除文件
// @Tags 文件管理
// @Security BearerAuth
// @Produce json
// @Param id path string true "文件 ID"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,403,404,503 {object} response.Body[any]
// @Router /sys/files/{id} [delete]
func (h *Handler) Remove(c *gin.Context) {
	id, err := httpx.ParseID(c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.Remove(c.Request.Context(), httpx.Actor(c), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// RegisterRoutes 分别要求上传、读取与删除权限。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, required gin.HandlerFunc, permit func(string) gin.HandlerFunc) {
	rg.Use(required)
	rg.POST("/upload", permit("sys:file:add"), h.Upload)
	rg.GET("/:id/download", permit("sys:file:get"), h.Download)
	rg.DELETE("/:id", permit("sys:file:remove"), h.Remove)
}
