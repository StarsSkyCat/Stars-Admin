package auth

import (
	"strings"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	token "github.com/StarsSkyCat/Stars-Admin/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// Handler 只负责绑定与服务调用，不直接访问持久化连接。
type Handler struct{ service *Service }

// NewHandler 创建认证 HTTP 入口。
func NewHandler(service *Service) *Handler { return &Handler{service} }

// Required 验证 Access，并对首次改密仅放行三个个人接口。
func (h *Handler) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		fields := strings.Fields(c.GetHeader("Authorization"))
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
			response.Fail(c, response.ErrUnauthenticated)
			return
		}
		principal, claims, err := h.service.Authenticate(c.Request.Context(), fields[1])
		if err != nil {
			response.Fail(c, err)
			return
		}
		if principal.MustChangePassword {
			switch c.FullPath() {
			case "/api/sys/auth/profile", "/api/sys/auth/password", "/api/sys/auth/logout":
			default:
				response.Fail(c, response.ErrPasswordChange)
				return
			}
		}
		c.Set("principal", principal)
		c.Request = c.Request.WithContext(security.WithPrincipal(c.Request.Context(), principal))
		c.Set("claims", claims)
		h.service.Touch(c.Request.Context(), principal, claims, c.ClientIP(), c.GetHeader("User-Agent"))
		c.Next()
	}
}

// Permit 执行显式路由权限和无归属映射资源的 admin 门禁。
func (h *Handler) Permit(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.service.Permit(c.Request.Context(), httpx.Actor(c), permission); err != nil {
			response.Fail(c, err)
			return
		}
		c.Next()
	}
}

// Captcha 生成一次性验证码。
// @Summary 获取登录验证码
// @Tags 认证
// @Produce json
// @Success 200 {object} response.Body[Captcha]
// @Failure 429,503 {object} response.Body[any]
// @Router /sys/auth/captcha [get]
func (h *Handler) Captcha(c *gin.Context) {
	out, err := h.service.Captcha(c.Request.Context(), c.ClientIP())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Login 登录并建立原子会话。
// @Summary 登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body LoginInput true "登录信息"
// @Success 200 {object} response.Body[token.Tokens]
// @Failure 400,401,403,429,503 {object} response.Body[any]
// @Router /sys/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var in LoginInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Login(c.Request.Context(), in, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Refresh 轮换单次 Refresh。
// @Summary 刷新双令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body RefreshInput true "刷新令牌"
// @Success 200 {object} response.Body[token.Tokens]
// @Failure 400,401,429,503 {object} response.Body[any]
// @Router /sys/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var in RefreshInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	out, err := h.service.Refresh(c.Request.Context(), in.RefreshToken, c.ClientIP())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Logout 撤销当前会话。
// @Summary 登出当前会话
// @Tags 认证
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[response.Status]
// @Failure 401,503 {object} response.Body[any]
// @Router /sys/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	value, _ := c.Get("claims")
	claims, ok := value.(*token.Claims)
	if !ok {
		response.Fail(c, response.ErrUnauthenticated)
		return
	}
	if err := h.service.Logout(c.Request.Context(), httpx.Actor(c), claims); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// Profile 返回本人当前权限。
// @Summary 获取本人信息及权限
// @Tags 认证
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Body[Profile]
// @Failure 401,503 {object} response.Body[any]
// @Router /sys/auth/profile [get]
func (h *Handler) Profile(c *gin.Context) {
	out, err := h.service.Profile(c.Request.Context(), httpx.Actor(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, out)
}

// Password 修改后撤销本人所有会话。
// @Summary 修改本人密码并下线
// @Tags 认证
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body PasswordInput true "旧密码与新密码"
// @Success 200 {object} response.Body[response.Status]
// @Failure 400,401,503 {object} response.Body[any]
// @Router /sys/auth/password [put]
func (h *Handler) Password(c *gin.Context) {
	var in PasswordInput
	if err := httpx.Bind(c, &in); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), httpx.Actor(c), in); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Status{Status: "ok"})
}

// RegisterRoutes 显式划分公开入口与受保护的个人入口。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/captcha", h.Captcha)
	rg.POST("/login", h.Login)
	rg.POST("/refresh", h.Refresh)
	private := rg.Group("", h.Required())
	private.POST("/logout", h.Logout)
	private.GET("/profile", h.Profile)
	private.PUT("/password", h.Password)
}
