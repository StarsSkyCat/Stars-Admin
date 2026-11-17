package app

import (
	"context"

	_ "github.com/StarsSkyCat/Stars-Admin/docs/swagger"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/middleware"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// Module 是装配契约，模块实现无需 import app。
type Module interface {
	Name() string
	RegisterRoutes(*gin.RouterGroup)
}

// NewRouter 装配基础路由并注入就绪检查，方便独立验证 HTTP 行为。
func NewRouter(cfg *config.Config, log *zap.Logger, cache middleware.Evaler, ready func(context.Context) error, modules ...Module) (*gin.Engine, error) {
	router := gin.New()
	if err := router.SetTrustedProxies(cfg.App.TrustedProxies); err != nil {
		return nil, err
	}
	router.HandleMethodNotAllowed = true
	router.Use(middleware.RequestID(), middleware.AccessLog(log), middleware.Recovery(log), middleware.CORS(cfg.CORS.Origins), middleware.RateLimit(cache, cfg.RateLimit, cfg.Redis, log))
	router.NoRoute(func(c *gin.Context) { response.Fail(c, response.ErrNotFound) })
	router.NoMethod(func(c *gin.Context) { response.Fail(c, response.ErrMethod) })
	api := router.Group("/api")
	api.GET("/ping", ping)
	api.GET("/health/live", live)
	health := healthHandler{check: ready, log: log}
	api.GET("/health/ready", health.ready)
	api.GET("/health", health.ready)
	for _, module := range modules {
		module.RegisterRoutes(api.Group("/" + module.Name()))
	}
	if cfg.Swagger.Enabled {
		router.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
	}
	return router, nil
}

// ping 检查基础 HTTP 链路。
// @Summary 检查 HTTP 连通性
// @Tags 基础设施
// @Produce json
// @Success 200 {object} response.Body[response.Status]
// @Failure 429 {object} response.Body[any]
// @Failure 503 {object} response.Body[any]
// @Router /ping [get]
func ping(c *gin.Context) { response.OK(c, response.Status{Status: "ok"}) }

// live 只反映进程存活，不访问外部服务。
// @Summary 检查进程存活
// @Tags 基础设施
// @Produce json
// @Success 200 {object} response.Body[response.Status]
// @Router /health/live [get]
func live(c *gin.Context) { response.OK(c, response.Status{Status: "alive"}) }

type healthHandler struct {
	check func(context.Context) error
	log   *zap.Logger
}

// ready 检查注入的就绪能力，不对外输出具体依赖错误。
// @Summary 检查服务就绪
// @Tags 基础设施
// @Produce json
// @Success 200 {object} response.Body[response.Status]
// @Failure 503 {object} response.Body[any]
// @Router /health/ready [get]
// @Router /health [get]
func (h healthHandler) ready(c *gin.Context) {
	if err := h.check(c.Request.Context()); err != nil {
		h.log.Warn("服务尚未就绪", zap.String("traceId", c.GetString("requestID")))
		response.Fail(c, response.ErrUnavailable)
		return
	}
	response.OK(c, response.Status{Status: "ready"})
}
