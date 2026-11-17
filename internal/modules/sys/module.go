// Package sys 装配系统功能与安全生命周期，避免功能直接访问其他功能持久化模型。
package sys

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/cache"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/middleware"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/permission"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/session"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/audit"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/auth"
	sysconfig "github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/dept"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/dict"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/file"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/menu"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/post"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/role"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/seed"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module 持有功能服务及必须在基础设施之前关闭的安全协调器。
type Module struct {
	authentication *auth.Handler
	users          *user.Handler
	roles          *role.Handler
	menus          *menu.Handler
	depts          *dept.Handler
	security       *security.Manager
	timeout        time.Duration
	posts          *post.Handler
	configs        *sysconfig.Handler
	dicts          *dict.Handler
	files          *file.Handler
	audits         *audit.Handler
	auditService   *audit.Service
	fileService    *file.Service
	duplicate      gin.HandlerFunc
	resolver       security.Resolver
}

type features struct {
	users *user.Service
	roles *role.Service
	menus *menu.Service
	depts *dept.Service
}

func assemble(deps core.Deps, gate *security.Manager) features {
	depts := dept.New(deps.DB, deps.IDs, gate)
	menus := menu.New(deps.DB, deps.IDs, gate)
	roles := role.New(deps.DB, deps.IDs, gate, menus, depts)
	users := user.New(deps.DB, deps.IDs, gate, roles, depts, deps.Config.Auth.BcryptCost)
	depts.SetUsers(users)
	menus.SetRoles(roles)
	roles.SetUsers(users)
	f := features{users, roles, menus, depts}
	resolver := scopeResolver{f}
	users.SetScopeResolver(resolver)
	depts.SetScopeResolver(resolver)
	return f
}

func (f features) project(ctx context.Context, tx *gorm.DB) ([]permission.Rule, error) {
	active, err := f.roles.ActiveIDs(ctx, tx)
	if err != nil {
		return nil, err
	}
	roles := map[int64]bool{}
	for _, id := range active {
		roles[int64(id)] = true
	}
	members, err := f.users.Memberships(ctx, tx)
	if err != nil {
		return nil, err
	}
	grants, err := f.roles.Grants(ctx, tx)
	if err != nil {
		return nil, err
	}
	perms, err := f.menus.Permissions(ctx, tx)
	if err != nil {
		return nil, err
	}
	rules := []permission.Rule{}
	seen := map[string]bool{}
	add := func(kind, left, right string) {
		key := kind + "\x00" + left + "\x00" + right
		if !seen[key] {
			seen[key] = true
			rules = append(rules, permission.Rule{Ptype: kind, V0: left, V1: right})
		}
	}
	for _, member := range members {
		if roles[member.RoleID] {
			add("g", "user:"+strconv.FormatInt(member.UserID, 10), "role:"+strconv.FormatInt(member.RoleID, 10))
		}
	}
	for _, grant := range grants {
		if roles[grant.RoleID] {
			if perm := perms[model.ID(grant.MenuID)]; perm != "" {
				add("p", "role:"+strconv.FormatInt(grant.RoleID, 10), perm)
			}
		}
	}
	return rules, nil
}

// New 在注册路由前完成投影重建和认证基础件初始化。
func New(ctx context.Context, deps core.Deps) (*Module, error) {
	policies, err := permission.New(deps.DB, deps.Config.Database.Timeout)
	if err != nil {
		return nil, err
	}
	sessions := session.New(deps.Redis, deps.Config.Redis.KeyPrefix, deps.Config.Redis.Timeout)
	gate := security.New(deps.DB, policies, sessions, deps.Logger, deps.Config.Database.Timeout)
	f := assemble(deps, gate)
	loginAudit := audit.New(deps.DB, deps.IDs)
	settings := sysconfig.New(deps.DB)
	caches := cache.New(deps.Redis, deps.Config.Redis.KeyPrefix+"cache:", deps.Logger)
	settings.Configure(caches, deps.IDs)
	dictionaries := dict.New(deps.DB, deps.IDs, caches)
	posts := post.New(deps.DB, deps.IDs, gate, f.users)
	f.users.SetPosts(posts)
	storage, err := file.NewLocal(deps.Config.File.LocalDir)
	if err != nil {
		return nil, err
	}
	files := file.New(deps.DB, storage, deps.IDs, deps.Config.File, deps.Logger)
	loginAudit.Configure(sessions, f.users)
	authentication, err := auth.New(deps.Config, deps.Redis, f.users, settings, loginAudit, gate, sessions, deps.Logger)
	if err != nil {
		_ = files.Close()
		return nil, err
	}
	if err := gate.Start(ctx, f.project); err != nil {
		_ = files.Close()
		_ = gate.Close()
		return nil, err
	}
	loginAudit.Start(deps.Logger)
	return &Module{authentication: auth.NewHandler(authentication), users: user.NewHandler(f.users), roles: role.NewHandler(f.roles), menus: menu.NewHandler(f.menus), depts: dept.NewHandler(f.depts), security: gate, timeout: deps.Config.Database.Timeout, posts: post.NewHandler(posts), configs: sysconfig.NewHandler(settings), dicts: dict.NewHandler(dictionaries), files: file.NewHandler(files, deps.Config.File.MaxSize), audits: audit.NewHandler(loginAudit), auditService: loginAudit, fileService: files, duplicate: middleware.DuplicateSubmit(deps.Redis, deps.Config.Redis.KeyPrefix), resolver: scopeResolver{f}}, nil
}

// Name 返回系统模块路由前缀。
func (*Module) Name() string { return "sys" }

// RegisterRoutes 按显式公开入口和逐动作权限进行装配。
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Use(requestDeadline(m.timeout), middleware.OperLog(m.auditService))
	m.authentication.RegisterRoutes(rg.Group("/auth"))
	m.users.RegisterRoutes(rg.Group("/users"), m.authentication.Required(), m.authentication.Permit, m.duplicate)
	m.roles.RegisterRoutes(rg.Group("/roles"), m.authentication.Required(), m.authentication.Permit)
	m.menus.RegisterRoutes(rg.Group("/menus"), m.authentication.Required(), m.authentication.Permit)
	m.depts.RegisterRoutes(rg.Group("/depts"), m.authentication.Required(), m.authentication.Permit)
	m.posts.RegisterRoutes(rg.Group("/posts"), m.authentication.Required(), m.authentication.Permit)
	m.configs.RegisterRoutes(rg.Group("/configs"), m.authentication.Required(), m.authentication.Permit)
	m.dicts.RegisterRoutes(rg.Group("/dict"), m.authentication.Required(), m.authentication.Permit)
	m.files.RegisterRoutes(rg.Group("/files"), m.authentication.Required(), m.authentication.Permit)
	m.audits.RegisterRoutes(rg.Group("/audit"), m.authentication.Required(), m.authentication.Permit)
}

func requestDeadline(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout(c.FullPath(), timeout))
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// Ready 让应用探针感知策略重载与撤销故障。
func (m *Module) Ready(ctx context.Context) error { return m.security.Ready(ctx) }

// Close 必须在 HTTP 在途请求排空后、数据库关闭前执行。
func (m *Module) Close() error {
	return errors.Join(m.auditService.Close(), m.fileService.Close(), m.security.Close())
}

// Seed 仅依赖数据库和哈希配置，不连接 Redis、不启动安全重试或 HTTP。
func Seed(ctx context.Context, deps core.Deps, password string) error {
	f := assemble(deps, nil)
	policies, err := permission.New(deps.DB, deps.Config.Database.Timeout)
	if err != nil {
		return err
	}
	extras := func(ctx context.Context, tx *gorm.DB) error {
		if err := sysconfig.New(deps.DB).Seed(ctx, tx, deps.IDs); err != nil {
			return err
		}
		return dict.Seed(ctx, tx, deps.IDs)
	}
	return seed.Run(ctx, deps.DB, f.depts, f.menus, f.roles, f.users, password, extras, f.project, policies)
}

func requestTimeout(route string, normal time.Duration) time.Duration {
	if route == "/api/sys/users/import" {
		return 30 * time.Second
	}
	return normal
}

// Guards 向装配层提供认证与横切能力，其他模块无需导入 sys。
func (m *Module) Guards() httpx.Guards {
	return httpx.Guards{Required: m.authentication.Required(), Permit: m.authentication.Permit, Operations: middleware.OperLog(m.auditService), Duplicate: m.duplicate}
}

// Access 提供与系统授权变更共享的读门禁和权限能力，供 app 手工装配业务模块。
func (m *Module) Access() interface {
	Read(func() error) error
	Allow(security.Principal, string) error
} {
	return m.security
}

// ScopeResolver 提供实时核对持久身份与有效角色的业务范围解析能力。
func (m *Module) ScopeResolver() security.Resolver { return m.resolver }
