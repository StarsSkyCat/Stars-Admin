package middleware

import (
	"bytes"
	"io"
	"time"
	"stars-admin/internal/models"
	"stars-admin/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return ""
	})
}

// OperationLogger 操作日志中间件
func OperationLogger(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// 读取请求体
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 创建响应写入器
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 记录操作日志
		go func() {
			latency := time.Since(start).Milliseconds()
			
			// 获取用户信息
			var userID uint
			var username string
			if claims, exists := c.Get("user"); exists {
				if userClaims, ok := claims.(*utils.JWTClaims); ok {
					userID = userClaims.UserID
					username = userClaims.Username
				}
			}

			// 创建操作日志
			operationLog := models.OperationLog{
				UserID:    userID,
				Username:  username,
				Method:    c.Request.Method,
				Path:      c.Request.URL.Path,
				IP:        c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Status:    c.Writer.Status(),
				Latency:   latency,
				Request:   string(requestBody),
				Response:  blw.body.String(),
				CreatedAt: time.Now(),
			}

			// 保存到数据库
			db.Create(&operationLog)
		}()
	}
}

// bodyLogWriter 响应体写入器
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// ErrorHandler 错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
					"path":  c.Request.URL.Path,
					"method": c.Request.Method,
				}).Error("Panic recovered")
				
				c.JSON(500, gin.H{
					"code":    500,
					"message": "Internal Server Error",
					"data":    nil,
				})
				c.Abort()
			}
		}()
		
		c.Next()
	}
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RateLimiter 限流中间件
func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 这里可以实现基于Redis的限流逻辑
		// 暂时跳过
		c.Next()
	}
}