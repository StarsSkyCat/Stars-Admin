package user

import (
	"io"
	"net/http"
	"strings"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
)

// Import 导入 Excel，错误行明细不包含原始单元格内容。
// @Summary 批量导入用户
// @Tags 用户管理
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "XLSX，最多 5MB、1000 行；首行 username,nickname,password,deptId,roleIds,postIds,email,phone,gender"
// @Success 200 {object} response.Body[ImportResult]
// @Failure 400,401,403,409,503 {object} response.Body[any]
// @Router /sys/users/import [post]
func (h *Handler) Import(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 6*1024*1024)
	if err := c.Request.ParseMultipartForm(1024 * 1024); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	if c.Request.MultipartForm != nil {
		defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	}
	header, err := c.FormFile("file")
	if err != nil || !strings.HasSuffix(strings.ToLower(header.Filename), ".xlsx") {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	r, err := header.Open()
	if err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	defer func() { _ = r.Close() }()
	data, err := io.ReadAll(io.LimitReader(r, 5*1024*1024+1))
	if err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	out, err := h.service.Import(c.Request.Context(), httpx.Actor(c), data)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Export 导出当前操作者范围内的资料。
// @Summary 导出用户资料
// @Tags 用户管理
// @Security BearerAuth
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param username query string false "用户名过滤"
// @Param deptId query string false "部门过滤"
// @Success 200 {file} binary
// @Failure 400,401,403,503 {object} response.Body[any]
// @Router /sys/users/export [post]
func (h *Handler) Export(c *gin.Context) {
	var q Query
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.ErrInvalidParam)
		return
	}
	data, err := h.service.Export(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="users.xlsx"`)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
