//go:build integration

package app

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/StarsSkyCat/Stars-Admin/internal/core"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/database"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/session"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys"
	"github.com/StarsSkyCat/Stars-Admin/pkg/hash"
	token "github.com/StarsSkyCat/Stars-Admin/pkg/jwt"
	"github.com/StarsSkyCat/Stars-Admin/pkg/snowflake"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const firstPassword = "Initial-Password-123"
const changedPassword = "Changed-Password-456"

type apiReply struct {
	Code   int            `json:"code"`
	Msg    string         `json:"msg"`
	Data   jsontext.Value `json:"data"`
	status int
}

func (r apiReply) As[T any](t *testing.T) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(r.Data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func callAPI(t *testing.T, a *App, method, path string, body any, bearer string) apiReply {
	t.Helper()
	var raw []byte
	var err error
	if body != nil {
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
	request.RemoteAddr = "192.0.2.1:1234"
	request.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	recorder := httptest.NewRecorder()
	a.router.ServeHTTP(recorder, request)
	var reply apiReply
	if err := json.Unmarshal(recorder.Body.Bytes(), &reply); err != nil {
		t.Fatalf("非统一响应 %d: %s", recorder.Code, recorder.Body.String())
	}
	reply.status = recorder.Code
	return reply
}
func requireCode(t *testing.T, r apiReply, status, code int) {
	t.Helper()
	if r.status != status || r.Code != code {
		t.Fatalf("状态 %d/%d，期望 %d/%d（%s）", r.status, r.Code, status, code, r.Msg)
	}
}
func seedApp(t *testing.T, options ...func(*config.Config)) *App {
	t.Helper()
	cfg := integrationConfig(t)
	cfg.Auth.BcryptCost = 4
	cfg.RateLimit.Enabled = false
	for _, option := range options {
		option(cfg)
	}
	if err := database.Migrate(t.Context(), cfg.Database, "up"); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(t.Context(), cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ORM.Exec("TRUNCATE sys_dept,sys_user,sys_role,sys_menu,sys_dict_type,sys_config,sys_login_log,casbin_rule RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	ids, err := snowflake.New(cfg.ID.Node)
	if err != nil {
		t.Fatal(err)
	}
	if err := sys.Seed(t.Context(), core.Deps{Config: cfg, DB: db.ORM, IDs: ids, Logger: zap.NewNop()}, firstPassword); err != nil {
		t.Fatal(err)
	}
	if err := db.ORM.Exec("UPDATE sys_config SET value='false' WHERE key='sys.account.captchaEnabled'").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	a, err := New(t.Context(), cfg, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Close(); err != nil && !errors.Is(err, redis.ErrClosed) {
			t.Error(err)
		}
	})
	return a
}

func TestM2RoleAndAccountStateRevokeAllSessions(t *testing.T) {
	a := seedApp(t)
	admin := adminLogin(t, a)
	roleID := objectID(t, callAPI(t, a, "POST", "/api/sys/roles", map[string]string{"name": "成员", "code": "member"}, admin.AccessToken))
	userID := objectID(t, callAPI(t, a, "POST", "/api/sys/users", map[string]string{"username": "bob", "nickname": "成员", "password": firstPassword}, admin.AccessToken))
	assign := func() {
		t.Helper()
		requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+userID+"/roles", map[string]any{"roleIds": []string{roleID}}, admin.AccessToken), 200, 0)
	}
	assign()
	initial := loginUser(t, a, "bob", firstPassword)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/auth/password", map[string]string{"oldPassword": firstPassword, "newPassword": changedPassword}, initial.AccessToken), 200, 0)
	member := loginUser(t, a, "bob", changedPassword)
	assign()
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, member.AccessToken), 401, 2000)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": member.RefreshToken}, ""), 401, 2007)
	member = loginUser(t, a, "bob", changedPassword)
	// 菜单授权更新即刻反映在本人资料，无需应用重启。
	menuID := objectID(t, callAPI(t, a, "POST", "/api/sys/menus", map[string]any{"name": "成员按钮", "type": 3, "perm": "sys:demo:read", "visible": 1, "status": 1}, admin.AccessToken))
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/roles/"+roleID+"/menus", map[string]any{"menuIds": []string{menuID}}, admin.AccessToken), 200, 0)
	profile := callAPI(t, a, "GET", "/api/sys/auth/profile", nil, member.AccessToken)
	requireCode(t, profile, 200, 0)
	if !strings.Contains(string(profile.Data), "sys:demo:read") {
		t.Fatal("授权未即时生效")
	}
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/roles/"+roleID+"/menus", map[string]any{"menuIds": []string{}}, admin.AccessToken), 200, 0)
	profile = callAPI(t, a, "GET", "/api/sys/auth/profile", nil, member.AccessToken)
	requireCode(t, profile, 200, 0)
	if strings.Contains(string(profile.Data), "sys:demo:read") {
		t.Fatal("撤销权限仍然生效")
	}
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/roles/"+roleID+"/status", map[string]int{"status": 2}, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, member.AccessToken), 403, 2006)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": member.RefreshToken}, ""), 401, 2007)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/roles/"+roleID+"/status", map[string]int{"status": 1}, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, member.AccessToken), 401, 2000)
	member = loginUser(t, a, "bob", changedPassword)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+userID+"/status", map[string]int{"status": 2}, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, member.AccessToken), 403, 2006)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": member.RefreshToken}, ""), 401, 2007)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "bob", "password": changedPassword}, ""), 403, 2006)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+userID+"/status", map[string]int{"status": 1}, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, member.AccessToken), 401, 2000)
	// 无效批量成员导致整个操作失败，先前存在的用户必须仍可查询。
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/users/"+userID+",9223372036854775807", nil, admin.AccessToken), 404, 1001)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/users/"+userID, nil, admin.AccessToken), 200, 0)
}

func TestM2ExpiredAccessStillAllowsRefreshWithinOriginalDeadline(t *testing.T) {
	a := seedApp(t, func(c *config.Config) { c.JWT.AccessTTL = time.Second })
	pair := loginUser(t, a, "admin", firstPassword)
	manager := token.New(a.cfg.JWT.Secret, a.cfg.JWT.Issuer, a.cfg.JWT.AccessTTL)
	old, err := manager.Parse(pair.RefreshToken, "refresh")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, pair.AccessToken), 401, 2001)
	reply := callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": pair.RefreshToken}, "")
	requireCode(t, reply, 200, 0)
	next := reply.As[token.Tokens](t)
	fresh, err := manager.Parse(next.RefreshToken, "refresh")
	if err != nil || !fresh.ExpiresAt.Equal(old.ExpiresAt.Time) || fresh.SID != old.SID {
		t.Fatal("刷新延长了原会话期限")
	}
	if !next.MustChangePassword {
		t.Fatal("Access 过期后刷新绕过了改密限制")
	}
}
func loginUser(t *testing.T, a *App, name, password string) token.Tokens {
	t.Helper()
	reply := callAPI(t, a, "POST", "/api/sys/auth/login", map[string]any{"username": name, "password": password}, "")
	requireCode(t, reply, 200, 0)
	return reply.As[token.Tokens](t)
}
func adminLogin(t *testing.T, a *App) token.Tokens {
	t.Helper()
	first := loginUser(t, a, "ADMIN", firstPassword)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/auth/password", map[string]any{"oldPassword": firstPassword, "newPassword": changedPassword}, first.AccessToken), 200, 0)
	return loginUser(t, a, "admin", changedPassword)
}
func objectID(t *testing.T, r apiReply) string {
	t.Helper()
	requireCode(t, r, 200, 0)
	out := r.As[struct {
		ID string `json:"id"`
	}](t)
	if out.ID == "" {
		t.Fatal("缺少字符串 ID")
	}
	return out.ID
}

func TestM2SeedFirstLoginAndAdministrativeCRUD(t *testing.T) {
	a := seedApp(t)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/users", nil, ""), 401, 2000)
	first := loginUser(t, a, "admin", firstPassword)
	if !first.MustChangePassword {
		t.Fatal("首次登录未标记改密")
	}
	requireCode(t, callAPI(t, a, "GET", "/api/sys/users", nil, first.AccessToken), 403, 2008)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, first.AccessToken), 200, 0)
	refresh := callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": first.RefreshToken}, "")
	requireCode(t, refresh, 200, 0)
	first = refresh.As[token.Tokens](t)
	if !first.MustChangePassword {
		t.Fatal("Refresh 绕过首次改密")
	}
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/auth/password", map[string]string{"oldPassword": "wrong-password", "newPassword": changedPassword}, first.AccessToken), 401, 2002)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/auth/password", map[string]string{"oldPassword": firstPassword, "newPassword": changedPassword}, first.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, first.AccessToken), 401, 2000)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": first.RefreshToken}, ""), 401, 2007)
	admin := loginUser(t, a, "admin", changedPassword)
	deptID := objectID(t, callAPI(t, a, "POST", "/api/sys/depts", map[string]any{"name": "研发", "parentId": "0", "status": 1}, admin.AccessToken))
	childID := objectID(t, callAPI(t, a, "POST", "/api/sys/depts", map[string]any{"name": "研发子部门", "parentId": deptID, "status": 1}, admin.AccessToken))
	pageReply := callAPI(t, a, "GET", "/api/sys/depts?name=研发&pageSize=1&orderBy=id%20desc", nil, admin.AccessToken)
	requireCode(t, pageReply, 200, 0)
	page := pageReply.As[struct {
		List []struct {
			ID string `json:"id"`
		} `json:"list"`
		Total int64 `json:"total"`
	}](t)
	if page.Total != 2 || len(page.List) != 1 || page.List[0].ID != childID {
		t.Fatal("部门列表未按数据库分页/排序返回")
	}
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/depts/"+deptID, map[string]any{"name": "研发", "parentId": childID, "status": 1}, admin.AccessToken), 409, 1003)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/depts/"+deptID, nil, admin.AccessToken), 409, 1003)
	for _, path := range []string{"/api/sys/depts/tree", "/api/sys/depts/options", "/api/sys/menus/tree", "/api/sys/menus/options"} {
		requireCode(t, callAPI(t, a, "GET", path, nil, admin.AccessToken), 200, 0)
	}
	menuID := objectID(t, callAPI(t, a, "POST", "/api/sys/menus", map[string]any{"name": "测试按钮", "type": 3, "perm": "sys:demo:read", "status": 1, "visible": 1}, admin.AccessToken))
	roleID := objectID(t, callAPI(t, a, "POST", "/api/sys/roles", map[string]any{"name": "测试角色", "code": "worker"}, admin.AccessToken))
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/roles/"+roleID+"/menus", map[string]any{"menuIds": []string{menuID}}, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/roles/"+roleID+"/data-scope", map[string]any{"dataScope": 2, "deptIds": []string{deptID}}, admin.AccessToken), 200, 0)
	userID := objectID(t, callAPI(t, a, "POST", "/api/sys/users", map[string]any{"username": "ALICE", "nickname": "用户", "password": firstPassword, "deptId": deptID}, admin.AccessToken))
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+userID+"/roles", map[string]any{"roleIds": []string{roleID}}, admin.AccessToken), 200, 0)
	// 独立 grant 字段不能通过普通编辑入口注入。
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+userID, map[string]any{"username": "alice", "nickname": "用户", "roleIds": []string{"1"}}, admin.AccessToken), 400, 1000)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+userID+"/roles", map[string]any{"roleIds": []string{"1"}}, admin.AccessToken), 403, 2004)
	aliceFirst := loginUser(t, a, "alice", firstPassword)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/auth/password", map[string]string{"oldPassword": firstPassword, "newPassword": changedPassword}, aliceFirst.AccessToken), 200, 0)
	alice := loginUser(t, a, "alice", changedPassword)
	profile := callAPI(t, a, "GET", "/api/sys/auth/profile", nil, alice.AccessToken)
	requireCode(t, profile, 200, 0)
	if !strings.Contains(string(profile.Data), "sys:demo:read") {
		t.Fatal("角色授权未生效")
	}
	requireCode(t, callAPI(t, a, "GET", "/api/sys/users", nil, alice.AccessToken), 403, 2004)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/roles/"+roleID, nil, admin.AccessToken), 409, 1003)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/depts/"+deptID, nil, admin.AccessToken), 409, 1003)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+userID+"/password", map[string]string{"password": "Reset-Password-789"}, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, alice.AccessToken), 401, 2000)
	reset := loginUser(t, a, "alice", "Reset-Password-789")
	if !reset.MustChangePassword {
		t.Fatal("重置密码后未要求改密")
	}
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/logout", nil, reset.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": reset.RefreshToken}, ""), 401, 2007)
	for _, path := range []string{"/api/sys/users/1/status", "/api/sys/roles/1/status"} {
		requireCode(t, callAPI(t, a, "PUT", path, map[string]int{"status": 2}, admin.AccessToken), 403, 2004)
	}
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/1/roles", map[string]any{"roleIds": []string{}}, admin.AccessToken), 403, 2004)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/users/"+userID+",1", nil, admin.AccessToken), 403, 2004)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/users/"+userID, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/users/"+userID, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/roles/"+roleID, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/menus/"+menuID, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/depts/"+childID, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/depts/"+deptID, nil, admin.AccessToken), 200, 0)
	// 重复 seed 不重置密码、参数或已有授权。
	ids, err := snowflake.New(0)
	if err != nil {
		t.Fatal(err)
	}
	if err := sys.Seed(t.Context(), core.Deps{Config: a.cfg, DB: a.db.ORM, IDs: ids, Logger: zap.NewNop()}, "Ignored-Password-999"); err != nil {
		t.Fatal(err)
	}
	var encoded string
	if err := a.db.ORM.Table("sys_user").Select("password").Where("id = ?", 1).Scan(&encoded).Error; err != nil || !hash.Match(encoded, changedPassword) {
		t.Fatal("seed 重置了密码")
	}
}

func TestM2CaptchaLockoutAndDependencyFailure(t *testing.T) {
	a := seedApp(t)
	if err := a.db.ORM.Exec("UPDATE sys_config SET value='true' WHERE key='sys.account.captchaEnabled'").Error; err != nil {
		t.Fatal(err)
	}
	image := callAPI(t, a, "GET", "/api/sys/auth/captcha", nil, "")
	requireCode(t, image, 200, 0)
	var captchaImage struct {
		Image string `json:"image"`
		UUID  string `json:"uuid"`
	}
	if err := json.Unmarshal(image.Data, &captchaImage); err != nil || !strings.HasPrefix(captchaImage.Image, "data:image/png;base64,") {
		t.Fatal("图片验证码响应无效")
	}
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "admin", "password": firstPassword, "captchaUuid": captchaImage.UUID, "captchaCode": "invalid"}, ""), 400, 2005)
	if a.cache.Exists(t.Context(), a.cfg.Redis.KeyPrefix+"captcha:"+captchaImage.UUID).Val() != 0 {
		t.Fatal("错误验证码也必须一次性消费")
	}
	id := uuid.New().String()
	if err := a.cache.Set(t.Context(), a.cfg.Redis.KeyPrefix+"captcha:"+id, session.Digest("123456"), time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	good := callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "admin", "password": firstPassword, "captchaUuid": id, "captchaCode": "123456"}, "")
	requireCode(t, good, 200, 0)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "admin", "password": firstPassword, "captchaUuid": id, "captchaCode": "123456"}, ""), 400, 2005)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "admin", "password": firstPassword, "captchaUuid": uuid.New().String(), "captchaCode": "123456"}, ""), 400, 2005)
	if err := a.db.ORM.Exec("UPDATE sys_config SET value='false' WHERE key='sys.account.captchaEnabled'").Error; err != nil {
		t.Fatal(err)
	}
	for i := range 5 {
		r := callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "missing", "password": "wrong-password"}, "")
		if i < 4 {
			requireCode(t, r, 401, 2002)
		} else {
			requireCode(t, r, 403, 2003)
		}
	}
	ttl := a.cache.TTL(t.Context(), a.cfg.Redis.KeyPrefix+"auth:fail:missing").Val()
	if ttl < 9*time.Minute || ttl > 10*time.Minute {
		t.Fatal("锁定 TTL 错误")
	}
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "missing", "password": firstPassword}, ""), 403, 2003)
	var count int64
	if err := a.db.ORM.Table("sys_login_log").Count(&count).Error; err != nil || count != 10 {
		t.Fatalf("登录日志数量 %d，%v", count, err)
	}
	for range 4 {
		requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "missing", "password": firstPassword}, ""), 403, 2003)
	}
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "missing", "password": firstPassword}, ""), 429, 1004)
	if err := a.cache.Close(); err != nil {
		t.Fatal(err)
	}
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": "admin", "password": firstPassword}, ""), 503, 1501)
	requireCode(t, callAPI(t, a, "GET", "/api/health/ready", nil, ""), 503, 1501)
	requireCode(t, callAPI(t, a, "GET", "/api/health/live", nil, ""), 200, 0)
}

func TestM2ConcurrentRefreshAndAccountRevocation(t *testing.T) {
	a := seedApp(t)
	admin := adminLogin(t, a)
	var success, refused atomic.Int64
	var wg sync.WaitGroup
	var next token.Tokens
	var mu sync.Mutex
	for range 20 {
		wg.Go(func() {
			r := callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": admin.RefreshToken}, "")
			if r.Code == 0 {
				success.Add(1)
				mu.Lock()
				next = r.As[token.Tokens](t)
				mu.Unlock()
			} else if r.Code == 2007 {
				refused.Add(1)
			} else {
				t.Errorf("并发刷新意外错误 %d", r.Code)
			}
		})
	}
	wg.Wait()
	if success.Load() != 1 || refused.Load() != 19 {
		t.Fatalf("轮换结果 %d/%d", success.Load(), refused.Load())
	}
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, admin.AccessToken), 401, 2000)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, next.AccessToken), 200, 0)
	// Access/Refresh 用途混淆必须被拒绝。
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, next.RefreshToken), 401, 2000)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": next.AccessToken}, ""), 401, 2007)
	// 账号不存在和错误密码都使用同一个公开码。
	for _, name := range []string{"admin", "unknown"} {
		requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/login", map[string]string{"username": name, "password": "wrong-password"}, ""), 401, 2002)
	}
}
