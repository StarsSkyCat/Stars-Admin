package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/session"
	sysconfig "github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/config"
	"github.com/StarsSkyCat/Stars-Admin/pkg/captcha"
	"github.com/StarsSkyCat/Stars-Admin/pkg/hash"
	token "github.com/StarsSkyCat/Stars-Admin/pkg/jwt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Users 是认证消费方定义的小接口，不依赖用户模型或 HTTP DTO。
type Users interface {
	ByUsername(context.Context, string) (security.Account, error)
	Account(context.Context, model.ID) (security.Account, error)
	Permissions(context.Context, []model.ID) ([]string, error)
	ChangePassword(context.Context, security.Principal, string, string) error
	RecordLogin(context.Context, model.ID, string) error
}

// Settings 提供已校验的只读认证参数。
type Settings interface {
	Read(context.Context) (sysconfig.Settings, error)
}

// Audit 通过注入能力记录事件，不让框架依赖业务日志表。
type Audit interface {
	Login(context.Context, string, string, bool, string) error
}
type gate interface {
	Read(func() error) error
	Allow(security.Principal, string) error
}

// Service 持有会话、认证快照与业务规则。
type Service struct {
	repo       repository
	users      Users
	settings   Settings
	audit      Audit
	gate       gate
	tokens     *token.Manager
	sessions   *session.Store
	refreshTTL time.Duration
	dummyHash  string
	log        *zap.Logger
}

// New 构造认证服务，预生成未知账号的等价 bcrypt 比对成本。
func New(cfg *config.Config, cache *redis.Client, users Users, settings Settings, audit Audit, gate gate, sessions *session.Store, log *zap.Logger) (*Service, error) {
	dummy, err := hash.Password(uuid.New().String(), cfg.Auth.BcryptCost)
	if err != nil {
		return nil, err
	}
	return &Service{repository{cache, cfg.Redis.KeyPrefix, cfg.Redis.Timeout}, users, settings, audit, gate, token.New(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.AccessTTL), sessions, cfg.JWT.RefreshTTL, dummy, log}, nil
}

func (s *Service) settingsFor(ctx context.Context) (sysconfig.Settings, error) {
	settings, err := s.settings.Read(ctx)
	if err != nil {
		s.log.Error("认证参数读取或类型校验失败")
		return settings, response.ErrUnavailable
	}
	return settings, nil
}
func (s *Service) limit(ctx context.Context, subject string, count int) error {
	ok, err := s.repo.rate(ctx, subject, count)
	if err != nil {
		return response.ErrUnavailable
	}
	if !ok {
		return response.ErrRateLimited
	}
	return nil
}
func (s *Service) record(ctx context.Context, name, ip string, err error) {
	message := "登录成功"
	if err != nil {
		business, ok := errors.AsType[*response.BizError](err)
		if ok {
			message = business.Msg
		} else {
			message = "登录失败"
		}
	}
	if e := s.audit.Login(ctx, name, ip, err == nil, message); e != nil {
		s.log.Error("登录日志写入失败")
	}
}

// Captcha 原子消费在登录执行，生成图片时不能忽略 Redis 错误。
func (s *Service) Captcha(ctx context.Context, ip string) (Captcha, error) {
	if err := s.limit(ctx, "captcha:ip:"+session.Digest(ip), 30); err != nil {
		return Captcha{}, err
	}
	settings, err := s.settingsFor(ctx)
	if err != nil {
		return Captcha{}, err
	}
	if !settings.CaptchaEnabled {
		return Captcha{Enabled: false}, nil
	}
	answer, image, err := captcha.Generate()
	if err != nil {
		return Captcha{}, err
	}
	id := uuid.New().String()
	if err := s.repo.putCaptcha(ctx, id, answer); err != nil {
		return Captcha{}, response.ErrUnavailable
	}
	return Captcha{true, id, image, 120}, nil
}

// Login 对未知账号与错误密码返回同一个错误，成功后才返回原子写入的令牌。
func (s *Service) Login(ctx context.Context, in LoginInput, ip string, userAgent ...string) (out token.Tokens, err error) {
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	defer func() { s.record(ctx, in.Username, ip, err) }()
	agent := ""
	if len(userAgent) > 0 {
		agent = userAgent[0]
	}
	if err = s.limit(ctx, "login:ip:"+session.Digest(ip), 20); err != nil {
		return out, err
	}
	err = s.gate.Read(func() error {
		settings, e := s.settingsFor(ctx)
		if e != nil {
			return e
		}
		if settings.CaptchaEnabled {
			if _, e := uuid.Parse(in.CaptchaUUID); e != nil {
				return response.ErrCaptcha
			}
			ok, e := s.repo.captcha(ctx, in.CaptchaUUID, in.CaptchaCode)
			if e != nil {
				return response.ErrUnavailable
			}
			if !ok {
				return response.ErrCaptcha
			}
		}
		if e := s.limit(ctx, "login:name:"+session.Digest(in.Username), 10); e != nil {
			return e
		}
		locked, e := s.repo.locked(ctx, in.Username, settings.PasswordRetryLimit)
		if e != nil {
			return response.ErrUnavailable
		}
		if locked {
			return response.ErrAccountLocked
		}
		account, e := s.users.ByUsername(ctx, in.Username)
		if e != nil && !errors.Is(e, response.ErrNotFound) {
			return response.ErrUnavailable
		}
		encoded := s.dummyHash
		if e == nil {
			encoded = account.Password
		}
		match := hash.Match(encoded, in.Password)
		if e != nil || !match {
			locked, e := s.repo.fail(ctx, in.Username, settings.PasswordRetryLimit, settings.LockMinutes)
			if e != nil {
				return response.ErrUnavailable
			}
			if locked {
				return response.ErrAccountLocked
			}
			return response.ErrCredentials
		}
		if account.Status != 1 || len(account.RoleIDs) == 0 {
			return response.ErrAccountDisabled
		}
		candidate, e := s.tokens.Issue(int64(account.ID), account.Username, "", time.Now().Add(s.refreshTTL), account.MustChangePassword)
		if e != nil {
			return e
		}
		if e = s.sessions.Login(ctx, int64(account.ID), account.AuthVersion, candidate, settings.MaxSessions, settings.MultiLogin); e != nil {
			return response.ErrUnavailable
		}
		if e = s.repo.clearFailures(ctx, in.Username); e != nil {
			return response.ErrUnavailable
		}
		if e = s.users.RecordLogin(ctx, account.ID, ip); e != nil {
			s.log.Error("最后登录元信息更新失败")
		}
		s.Touch(ctx, account.Principal, &candidate.Access, ip, agent)
		out = candidate
		return nil
	})
	return out, err
}

// Authenticate 验证令牌用途并核对数据库与 Redis 的当前状态。
func (s *Service) Authenticate(ctx context.Context, raw string) (principal security.Principal, claims *token.Claims, err error) {
	claims, err = s.tokens.Parse(raw, "access")
	if err != nil {
		if token.Expired(err) {
			return principal, nil, response.ErrTokenExpired
		}
		return principal, nil, response.ErrUnauthenticated
	}
	id, _ := strconv.ParseInt(claims.UserID, 10, 64)
	err = s.gate.Read(func() error {
		account, e := s.users.Account(ctx, model.ID(id))
		if e != nil {
			if errors.Is(e, response.ErrNotFound) {
				return response.ErrUnauthenticated
			}
			return response.ErrUnavailable
		}
		if account.Status != 1 || len(account.RoleIDs) == 0 {
			return response.ErrAccountDisabled
		}
		if account.Username != claims.Username {
			return response.ErrUnauthenticated
		}
		ok, e := s.sessions.Validate(ctx, id, account.AuthVersion, claims)
		if e != nil {
			return response.ErrUnavailable
		}
		if !ok {
			return response.ErrUnauthenticated
		}
		principal = account.Principal
		return nil
	})
	return principal, claims, err
}

// Refresh 在读取当前账号后，用 Lua 原子消费当前 Refresh；不扩展绝对期限。
func (s *Service) Refresh(ctx context.Context, raw, ip string) (out token.Tokens, err error) {
	if err = s.limit(ctx, "refresh:ip:"+session.Digest(ip), 60); err != nil {
		return out, err
	}
	claims, err := s.tokens.Parse(raw, "refresh")
	if err != nil {
		return out, response.ErrRefresh
	}
	if err = s.limit(ctx, "refresh:sid:"+claims.SID, 30); err != nil {
		return out, err
	}
	id, _ := strconv.ParseInt(claims.UserID, 10, 64)
	err = s.gate.Read(func() error {
		account, e := s.users.Account(ctx, model.ID(id))
		if e != nil {
			if errors.Is(e, response.ErrNotFound) {
				return response.ErrRefresh
			}
			return response.ErrUnavailable
		}
		if account.Status != 1 || len(account.RoleIDs) == 0 || account.Username != claims.Username {
			return response.ErrRefresh
		}
		candidate, e := s.tokens.Issue(id, account.Username, claims.SID, claims.ExpiresAt.Time, account.MustChangePassword)
		if e != nil {
			return response.ErrRefresh
		}
		ok, e := s.sessions.Rotate(ctx, id, account.AuthVersion, claims, raw, candidate)
		if e != nil {
			return response.ErrUnavailable
		}
		if !ok {
			return response.ErrRefresh
		}
		s.Touch(ctx, account.Principal, &candidate.Access, ip, "")
		out = candidate
		return nil
	})
	return out, err
}

// Logout 只撤销该次请求认证的当前 sid。
func (s *Service) Logout(ctx context.Context, p security.Principal, claims *token.Claims) error {
	return s.gate.Read(func() error {
		ok, err := s.sessions.Logout(ctx, int64(p.ID), p.AuthVersion, claims)
		if err != nil {
			return response.ErrUnavailable
		}
		if !ok {
			return response.ErrUnauthenticated
		}
		return nil
	})
}

// Profile 重新读取当前资料与权限，避免返回陈旧授权集合。
func (s *Service) Profile(ctx context.Context, p security.Principal) (out Profile, err error) {
	err = s.gate.Read(func() error {
		account, e := s.users.Account(ctx, p.ID)
		if e != nil {
			return response.ErrUnavailable
		}
		if account.AuthVersion != p.AuthVersion {
			return response.ErrUnauthenticated
		}
		perms, e := s.users.Permissions(ctx, account.RoleIDs)
		if e != nil {
			return response.ErrUnavailable
		}
		out = Profile{Principal: account.Principal, Nickname: account.Nickname, Permissions: perms}
		return nil
	})
	return out, err
}

// ChangePassword 委托用户业务执行哈希更新和全部会话撤销。
func (s *Service) ChangePassword(ctx context.Context, p security.Principal, in PasswordInput) error {
	return s.users.ChangePassword(ctx, p, in.OldPassword, in.NewPassword)
}

// Permit 在管理动作前重新核对身份版本及有效角色，关闭认证与授权之间的竞态窗口。
func (s *Service) Permit(ctx context.Context, p security.Principal, permission string) error {
	return s.gate.Read(func() error {
		account, err := s.users.Account(ctx, p.ID)
		if err != nil {
			return response.ErrUnavailable
		}
		if account.Status != 1 || account.AuthVersion != p.AuthVersion {
			return response.ErrUnauthenticated
		}
		if account.MustChangePassword {
			return response.ErrPasswordChange
		}
		if !account.Admin() && !scopedPermission(permission) {
			return response.ErrPermission
		}
		return s.gate.Allow(account.Principal, permission)
	})
}

func scopedPermission(perm string) bool {
	for _, prefix := range []string{"sys:user:", "sys:dept:", "sys:file:"} {
		if strings.HasPrefix(perm, prefix) {
			return true
		}
	}
	return false
}

// Touch 更新有效会话的白名单在线摘要，不延长令牌有效期。
func (s *Service) Touch(ctx context.Context, p security.Principal, claims *token.Claims, ip, agent string) {
	browser := "Other"
	for _, name := range []string{"Firefox", "Edg", "Chrome", "Safari"} {
		if strings.Contains(agent, name) {
			browser = name
			break
		}
	}
	if err := s.sessions.Touch(ctx, int64(p.ID), claims, p.Username, ip, browser); err != nil {
		s.log.Warn("在线摘要更新失败")
	}
}
