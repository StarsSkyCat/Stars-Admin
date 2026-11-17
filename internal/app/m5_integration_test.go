//go:build integration

package app

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
)

func TestM5AuthenticatedAccessAndRefreshRejectRedisOutage(t *testing.T) {
	// 取得当前空闲的本机地址，验证无法连接时的启动拒绝。
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	a := seedApp(t)
	pair := adminLogin(t, a)
	// 新建应用只替换 Redis 地址；同一持久身份不能因临时状态不可用而获放行。
	cfg := *a.cfg
	cfg.Redis.Addr = addr
	cfg.Redis.Timeout = 100 * time.Millisecond
	// 启动门禁必须拒绝不可达 Redis；已有应用故障另用关闭连接明确断言认证/刷新链路。
	if failed, err := New(t.Context(), &cfg, a.log); err == nil {
		_ = failed.Close()
		t.Fatal("Redis 不可达仍启动")
	}
	if err := a.cache.Close(); err != nil {
		t.Fatal(err)
	}
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, pair.AccessToken), 503, 1501)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/users", nil, pair.AccessToken), 503, 1501)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": pair.RefreshToken}, ""), 503, 1501)
	requireCode(t, callAPI(t, a, "GET", "/api/health/ready", nil, ""), 503, 1501)
	requireCode(t, callAPI(t, a, "GET", "/api/health/live", nil, ""), 200, 0)
}

func TestM5PostEditOptionsDeletionAndUniqueReservation(t *testing.T) {
	a := seedApp(t, func(c *config.Config) { c.Redis.Timeout = time.Second })
	admin := adminLogin(t, a)
	id := objectID(t, callAPI(t, a, "POST", "/api/sys/posts", map[string]any{"name": "测试岗位", "code": "m5_post", "status": 1}, admin.AccessToken))
	requireCode(t, callAPI(t, a, "GET", "/api/sys/posts/"+id, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/posts/"+id, map[string]any{"name": "停用岗位", "code": "m5_post", "status": 2}, admin.AccessToken), 200, 0)
	options := callAPI(t, a, "GET", "/api/sys/posts/options", nil, admin.AccessToken)
	requireCode(t, options, 200, 0)
	if strings.Contains(string(options.Data), id) {
		t.Fatal("停用岗位仍出现在可选项")
	}
	requireCode(t, callAPI(t, a, "GET", "/api/sys/posts?orderBy=unknown", nil, admin.AccessToken), 400, 1000)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/posts/"+id+",9223372036854775807", nil, admin.AccessToken), 404, 1001)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/posts/"+id, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/posts/"+id, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/posts", map[string]any{"name": "保留唯一键", "code": "m5_post", "status": 1}, admin.AccessToken), 409, 1003)
}

func TestM5RoleEditPermissionDoesNotOpenAdminOnlyAPI(t *testing.T) {
	a := seedApp(t)
	admin := adminLogin(t, a)
	role := objectID(t, callAPI(t, a, "POST", "/api/sys/roles", map[string]string{"name": "角色编辑持有者", "code": "m5_role_editor"}, admin.AccessToken))
	grantPermissions(t, a, admin, role, "sys:role:edit")
	_, member := createMember(t, a, admin, "m5_editor", "", role)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/roles/1", map[string]string{"name": "不得改名", "code": "admin"}, member.AccessToken), 403, 2004)
	unchanged := callAPI(t, a, "GET", "/api/sys/roles/1", nil, admin.AccessToken)
	requireCode(t, unchanged, 200, 0)
	if strings.Contains(string(unchanged.Data), "不得改名") {
		t.Fatal("普通角色绕过全局资源门禁")
	}
}
