package utils

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PageResponse 分页响应结构
type PageResponse struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 成功响应（带消息）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// BadRequest 请求参数错误
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    400,
		Message: message,
		Data:    nil,
	})
}

// Unauthorized 未授权
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    401,
		Message: message,
		Data:    nil,
	})
}

// Forbidden 禁止访问
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Code:    403,
		Message: message,
		Data:    nil,
	})
}

// NotFound 资源未找到
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Code:    404,
		Message: message,
		Data:    nil,
	})
}

// InternalServerError 服务器内部错误
func InternalServerError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Response{
		Code:    500,
		Message: message,
		Data:    nil,
	})
}

// PageSuccess 分页成功响应
func PageSuccess(c *gin.Context, list interface{}, total int64, page int, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: PageResponse{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// ValidateError 验证错误响应
func ValidateError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    400,
		Message: err.Error(),
		Data:    nil,
	})
}