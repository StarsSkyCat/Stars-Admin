//go:build integration

package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"gorm.io/gorm"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/audit"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/dict"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/sys/user"
	"github.com/StarsSkyCat/Stars-Admin/pkg/excel"
	token "github.com/StarsSkyCat/Stars-Admin/pkg/jwt"
)

func uploadAPI(t *testing.T, a *App, path, name string, data []byte, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	f, err := w.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", path, &body)
	r.RemoteAddr = "192.0.2.1:1234"
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.Header.Set("Authorization", "Bearer "+bearer)
	out := httptest.NewRecorder()
	a.router.ServeHTTP(out, r)
	return out
}
func replyOf(t *testing.T, r *httptest.ResponseRecorder) apiReply {
	t.Helper()
	var out apiReply
	if err := json.Unmarshal(r.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	out.status = r.Code
	return out
}
func binaryAPI(t *testing.T, a *App, method, path, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, nil)
	r.RemoteAddr = "192.0.2.1:1234"
	r.Header.Set("Authorization", "Bearer "+bearer)
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	return w
}
func createMember(t *testing.T, a *App, admin token.Tokens, name, dept, role string) (string, token.Tokens) {
	t.Helper()
	in := map[string]any{"username": name, "nickname": name, "password": firstPassword}
	if dept != "" {
		in["deptId"] = dept
	}
	id := objectID(t, callAPI(t, a, "POST", "/api/sys/users", in, admin.AccessToken))
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+id+"/roles", map[string]any{"roleIds": []string{role}}, admin.AccessToken), 200, 0)
	pair := loginUser(t, a, name, firstPassword)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/auth/password", map[string]string{"oldPassword": firstPassword, "newPassword": changedPassword}, pair.AccessToken), 200, 0)
	return id, loginUser(t, a, name, changedPassword)
}
func grantPermissions(t *testing.T, a *App, admin token.Tokens, role string, perms ...string) {
	t.Helper()
	ids := []string{}
	for i, perm := range perms {
		ids = append(ids, objectID(t, callAPI(t, a, "POST", "/api/sys/menus", map[string]any{"name": fmt.Sprintf("验收按钮%d", i), "type": 3, "perm": perm, "visible": 1, "status": 1}, admin.AccessToken)))
	}
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/roles/"+role+"/menus", map[string]any{"menuIds": ids}, admin.AccessToken), 200, 0)
}
func TestM3FiveScopesUnionAndObjectWriteBoundaries(t *testing.T) {
	a := seedApp(t)
	admin := adminLogin(t, a)
	for _, sql := range []string{"INSERT INTO sys_dept(id,name,ancestors,parent_id) VALUES(10,'本部','0',0),(11,'子部','0,10',10),(100,'路径边界','0',0),(101,'另一子部','0,100',100),(20,'外部','0',0)"} {
		if err := a.db.ORM.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	role := objectID(t, callAPI(t, a, "POST", "/api/sys/roles", map[string]string{"name": "范围管理员", "code": "scope_manager"}, admin.AccessToken))
	grantPermissions(t, a, admin, role, "sys:user:list", "sys:user:get", "sys:user:edit", "sys:user:remove", "sys:user:export", "sys:user:import", "sys:user:grant", "sys:user:resetPwd", "sys:dept:list", "sys:dept:get", "sys:dept:edit", "sys:dept:add", "sys:dept:remove")
	actor, pair := createMember(t, a, admin, "scope_actor", "10", role)
	members := map[string]string{}
	for _, u := range []struct{ name, dept string }{{"alpha", "10"}, {"child", "11"}, {"boundary", "100"}, {"outer", "20"}} {
		members[u.name] = objectID(t, callAPI(t, a, "POST", "/api/sys/users", map[string]any{"username": u.name, "nickname": u.name, "password": firstPassword, "deptId": u.dept}, admin.AccessToken))
	}
	setScope := func(kind int, depts []string) {
		t.Helper()
		requireCode(t, callAPI(t, a, "PUT", "/api/sys/roles/"+role+"/data-scope", map[string]any{"dataScope": kind, "deptIds": depts}, admin.AccessToken), 200, 0)
	}
	for _, tc := range []struct {
		name        string
		kind        int
		departments []string
		want        []string
	}{{"全部", 1, []string{}, []string{"1", actor, members["alpha"], members["child"], members["boundary"], members["outer"]}}, {"自定义", 2, []string{"20"}, []string{members["outer"]}}, {"本部门", 3, []string{}, []string{actor, members["alpha"]}}, {"含子部", 4, []string{}, []string{actor, members["alpha"], members["child"]}}, {"仅本人", 5, []string{}, []string{actor}}} {
		t.Run(tc.name, func(t *testing.T) {
			setScope(tc.kind, tc.departments)
			r := callAPI(t, a, "GET", "/api/sys/users?pageSize=100", nil, pair.AccessToken)
			requireCode(t, r, 200, 0)
			page := r.As[response.Page[user.View]](t)
			got := []string{}
			for _, u := range page.List {
				got = append(got, fmt.Sprint(int64(u.ID)))
			}
			slices.Sort(got)
			want := slices.Clone(tc.want)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("范围不正确 got=%v want=%v", got, want)
			}
			w := binaryAPI(t, a, "POST", "/api/sys/users/export", pair.AccessToken)
			if w.Code != 200 {
				t.Fatalf("导出失败 %s", w.Body.String())
			}
			rows, err := excel.Read(w.Body.Bytes(), 10001)
			if err != nil || len(rows) != len(want)+1 {
				t.Fatalf("导出范围不同步 %v %d", err, len(rows))
			}
			if strings.Contains(strings.Join(rows[0], ","), "password") {
				t.Fatal("导出包含密码")
			}
		})
	}
	setScope(4, []string{})
	for _, path := range []string{"/api/sys/users/" + members["boundary"], "/api/sys/depts/100"} {
		requireCode(t, callAPI(t, a, "GET", path, nil, pair.AccessToken), 404, 1001)
	}
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/users/"+members["alpha"]+","+members["outer"], nil, pair.AccessToken), 404, 1001)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/users/"+members["alpha"], nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+members["outer"]+"/status", map[string]int{"status": 2}, pair.AccessToken), 404, 1001)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+members["outer"]+"/roles", map[string]any{"roleIds": []string{}}, pair.AccessToken), 404, 1001)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+members["outer"]+"/password", map[string]string{"password": firstPassword}, pair.AccessToken), 404, 1001)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+members["alpha"], map[string]any{"username": "alpha", "nickname": "alpha", "deptId": "20"}, pair.AccessToken), 403, 2004)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/depts/11", map[string]any{"name": "子部", "parentId": "20", "status": 1}, pair.AccessToken), 404, 1001)
	tree := callAPI(t, a, "GET", "/api/sys/depts/options", nil, pair.AccessToken)
	requireCode(t, tree, 200, 0)
	if strings.Contains(string(tree.Data), `"100"`) || !strings.Contains(string(tree.Data), `"11"`) {
		t.Fatal("部门选项范围泄漏或遗漏")
	}
	requireCode(t, callAPI(t, a, "GET", "/api/sys/roles", nil, pair.AccessToken), 403, 2004)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+actor+"/roles", map[string]any{"roleIds": []string{"1"}}, pair.AccessToken), 403, 2004)
	// 多角色并集：自定义外部部门与仅本人组合。
	setScope(2, []string{"20"})
	selfRole := objectID(t, callAPI(t, a, "POST", "/api/sys/roles", map[string]string{"name": "本人", "code": "self"}, admin.AccessToken))
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+actor+"/roles", map[string]any{"roleIds": []string{role, selfRole}}, admin.AccessToken), 200, 0)
	pair = loginUser(t, a, "scope_actor", changedPassword)
	r := callAPI(t, a, "GET", "/api/sys/users", nil, pair.AccessToken)
	requireCode(t, r, 200, 0)
	if page := r.As[response.Page[user.View]](t); page.Total != 2 {
		t.Fatal("多角色并集失败")
	}
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/depts/20", map[string]any{"name": "外部", "parentId": "0", "status": 2}, admin.AccessToken), 200, 0)
	r = callAPI(t, a, "GET", "/api/sys/users", nil, pair.AccessToken)
	requireCode(t, r, 200, 0)
	if r.As[response.Page[user.View]](t).Total != 1 {
		t.Fatal("停用自定义部门未排除")
	}
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+actor+"/roles", map[string]any{"roleIds": []string{role}}, admin.AccessToken), 200, 0)
	pair = loginUser(t, a, "scope_actor", changedPassword)
	r = callAPI(t, a, "GET", "/api/sys/users", nil, pair.AccessToken)
	requireCode(t, r, 200, 0)
	if r.As[response.Page[user.View]](t).Total != 0 {
		t.Fatal("自定义有效集合为空时未默认拒绝")
	}
}
func TestM3CachingPostsImportAndAudit(t *testing.T) {
	a := seedApp(t)
	admin := adminLogin(t, a)
	typ := objectID(t, callAPI(t, a, "POST", "/api/sys/dict/types", map[string]any{"name": "验收字典", "type": "m3_test", "status": 1}, admin.AccessToken))
	entry := objectID(t, callAPI(t, a, "POST", "/api/sys/dict/data", map[string]any{"dictType": "m3_test", "label": "第一版", "value": "a", "status": 1}, admin.AccessToken))
	lookup := func(label string) {
		t.Helper()
		r := callAPI(t, a, "GET", "/api/sys/dict/data/type/m3_test", nil, admin.AccessToken)
		requireCode(t, r, 200, 0)
		if rows := r.As[[]dict.DataView](t); len(rows) != 1 || rows[0].Label != label {
			t.Fatal("缓存字典文本未同步")
		}
	}
	lookup("第一版")
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/dict/data/"+entry, map[string]any{"dictType": "m3_test", "label": "第二版", "value": "a", "status": 1}, admin.AccessToken), 200, 0)
	lookup("第二版")
	requireCode(t, callAPI(t, a, "POST", "/api/sys/dict/cache/refresh", nil, admin.AccessToken), 200, 0)
	lookup("第二版")
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/dict/types/"+typ, map[string]any{"name": "验收字典", "type": "renamed", "status": 1}, admin.AccessToken), 400, 1000)
	configID := objectID(t, callAPI(t, a, "POST", "/api/sys/configs", map[string]string{"name": "标题", "key": "app.title", "value": "第一版"}, admin.AccessToken))
	requireCode(t, callAPI(t, a, "GET", "/api/sys/configs/key/app.title", nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/configs/"+configID, map[string]string{"name": "标题", "key": "app.title", "value": "第二版"}, admin.AccessToken), 200, 0)
	r := callAPI(t, a, "GET", "/api/sys/configs/key/app.title", nil, admin.AccessToken)
	requireCode(t, r, 200, 0)
	if !strings.Contains(string(r.Data), "第二版") {
		t.Fatal("参数缓存失效失败")
	}
	requireCode(t, callAPI(t, a, "POST", "/api/sys/configs/cache/refresh", nil, admin.AccessToken), 200, 0)
	var builtin string
	if err := a.db.ORM.Table("sys_config").Select("id").Where("key = ?", "sys.account.maxSessions").Scan(&builtin).Error; err != nil {
		t.Fatal(err)
	}
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/configs/"+builtin, nil, admin.AccessToken), 403, 2004)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/configs/"+builtin, map[string]string{"name": "会话", "key": "sys.account.maxSessions", "value": "0"}, admin.AccessToken), 400, 1000)
	post := objectID(t, callAPI(t, a, "POST", "/api/sys/posts", map[string]any{"code": "DEV", "name": "研发", "status": 1}, admin.AccessToken))
	role := objectID(t, callAPI(t, a, "POST", "/api/sys/roles", map[string]string{"code": "reader", "name": "读者"}, admin.AccessToken))
	uid, member := createMember(t, a, admin, "m3_reader", "", role)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/dict/data/type/m3_test", nil, member.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/configs/key/app.title", nil, member.AccessToken), 403, 2004)
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/audit/login-logs/clean", nil, member.AccessToken), 403, 2004)
	for _, ids := range [][]string{{post}, {}, {post}} {
		requireCode(t, callAPI(t, a, "PUT", "/api/sys/users/"+uid+"/posts", map[string]any{"postIds": ids}, admin.AccessToken), 200, 0)
	}
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/posts/"+post, nil, admin.AccessToken), 409, 1003)
	headers := []string{"username", "nickname", "password", "deptId", "roleIds", "postIds", "email", "phone", "gender"}
	makeSheet := func(rows ...[]string) []byte {
		t.Helper()
		raw, e := excel.Write(append([][]string{headers}, rows...))
		if e != nil {
			t.Fatal(e)
		}
		return raw
	}
	good := []string{"import_good", "导入成员", firstPassword, "", role, post, "", "", "0"}
	bad := []string{"import_bad", "错误成员", firstPassword, "9223372036854775807", role, post, "", "", "0"}
	r = replyOf(t, uploadAPI(t, a, "/api/sys/users/import", "users.xlsx", makeSheet(good, bad), admin.AccessToken))
	requireCode(t, r, 200, 0)
	result := r.As[user.ImportResult](t)
	if result.Imported != 0 || len(result.Errors) != 1 || result.Errors[0].Row != 3 {
		t.Fatal("整批拒绝或错误行报告失效")
	}
	var count int64
	if err := a.db.ORM.Table("sys_user").Where("username = ?", "import_good").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("错误行导致了部分写入")
	}
	r = replyOf(t, uploadAPI(t, a, "/api/sys/users/import", "users.xlsx", makeSheet(good), admin.AccessToken))
	requireCode(t, r, 200, 0)
	if r.As[user.ImportResult](t).Imported != 1 {
		t.Fatal("导入成功数错误")
	}
	initial := loginUser(t, a, "import_good", firstPassword)
	if !initial.MustChangePassword {
		t.Fatal("导入绕过首次改密")
	}
	r = replyOf(t, uploadAPI(t, a, "/api/sys/users/import", "users.xlsx", makeSheet(good), admin.AccessToken))
	requireCode(t, r, 200, 0)
	if r.As[user.ImportResult](t).Imported != 0 {
		t.Fatal("导入覆盖现有用户")
	}
	if w := binaryAPI(t, a, "POST", "/api/sys/audit/login-logs/export", admin.AccessToken); w.Code != 200 {
		t.Fatalf("登录日志导出失败 %s", w.Body.String())
	}
	online := callAPI(t, a, "GET", "/api/sys/audit/online", nil, admin.AccessToken)
	requireCode(t, online, 200, 0)
	if online.As[response.Page[audit.OnlineView]](t).Total < 2 {
		t.Fatal("有效会话未展示")
	}
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/audit/online/"+uid, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/auth/profile", nil, member.AccessToken), 401, 2000)
	requireCode(t, callAPI(t, a, "POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": member.RefreshToken}, ""), 401, 2007)
	// 操作日志异步写入后验证白名单，密码、令牌及 Excel 单元格均不落原文。
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var n int64
		if err := a.db.ORM.Table("sys_oper_log").Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	logs := callAPI(t, a, "GET", "/api/sys/audit/oper-logs?pageSize=100", nil, admin.AccessToken)
	requireCode(t, logs, 200, 0)
	if logs.As[response.Page[audit.OperView]](t).Total == 0 {
		t.Fatal("写请求未产生审计日志")
	}
	for _, secret := range []string{firstPassword, changedPassword, admin.AccessToken, "import_good"} {
		if strings.Contains(string(logs.Data), secret) {
			t.Fatal("审计泄漏请求或认证原文")
		}
	}
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/dict/types/"+typ, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/dict/data/"+entry, nil, admin.AccessToken), 404, 1001)
	requireCode(t, callAPI(t, a, "GET", "/api/sys/dict/data/type/m3_test", nil, admin.AccessToken), 404, 1001)
}

func TestM3FileOwnershipDedupConcurrencyAndDatabaseFailure(t *testing.T) {
	a := seedApp(t, func(c *config.Config) { c.File.MaxSize = 1024 })
	admin := adminLogin(t, a)
	role := objectID(t, callAPI(t, a, "POST", "/api/sys/roles", map[string]string{"name": "文件成员", "code": "files"}, admin.AccessToken))
	grantPermissions(t, a, admin, role, "sys:file:add", "sys:file:get", "sys:file:remove")
	_, member := createMember(t, a, admin, "file_member", "", role)
	payload := []byte("file verification content")
	owner := objectID(t, replyOf(t, uploadAPI(t, a, "/api/sys/files/upload", "a.txt", payload, admin.AccessToken)))
	other := objectID(t, replyOf(t, uploadAPI(t, a, "/api/sys/files/upload", "b.txt", payload, member.AccessToken)))
	if owner == other {
		t.Fatal("跨所有者复用了实体")
	}
	if duplicate := objectID(t, replyOf(t, uploadAPI(t, a, "/api/sys/files/upload", "copy.txt", payload, member.AccessToken))); duplicate != other {
		t.Fatal("同所有者去重失效")
	}
	for _, method := range []string{"GET", "DELETE"} {
		path := "/api/sys/files/" + owner
		if method == "GET" {
			path += "/download"
		}
		requireCode(t, replyOf(t, binaryAPI(t, a, method, path, member.AccessToken)), 403, 2004)
	}
	download := binaryAPI(t, a, "GET", "/api/sys/files/"+other+"/download", member.AccessToken)
	if download.Code != 200 || !bytes.Equal(download.Body.Bytes(), payload) || download.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(download.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal("受保护附件响应不正确")
	}
	for _, in := range []struct {
		name string
		body []byte
	}{{"fake.png", payload}, {"huge.txt", bytes.Repeat([]byte("a"), 1025)}, {"bad.exe", payload}} {
		requireCode(t, replyOf(t, uploadAPI(t, a, "/api/sys/files/upload", in.name, in.body, member.AccessToken)), 400, 1000)
	}
	var wg sync.WaitGroup
	var failed atomic.Int64
	for range 8 {
		wg.Go(func() {
			r := replyOf(t, uploadAPI(t, a, "/api/sys/files/upload", "same.txt", payload, member.AccessToken))
			if r.status != 200 || r.Code != 0 {
				failed.Add(1)
			}
		})
	}
	wg.Wait()
	if failed.Load() != 0 {
		t.Fatal("并发去重失败")
	}
	var count int64
	if err := a.db.ORM.Table("sys_file").Where("deleted_at IS NULL").Count(&count).Error; err != nil || count != 2 {
		t.Fatal("并发上传留下重复记录")
	}
	before, err := os.ReadDir(a.cfg.File.LocalDir)
	if err != nil {
		t.Fatal(err)
	}
	callbackName := "m3:file:create-failure"
	if err := a.db.ORM.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_file" {
			tx.AddError(errors.New("模拟数据库故障"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	r := replyOf(t, uploadAPI(t, a, "/api/sys/files/upload", "failure.txt", []byte("new content"), member.AccessToken))
	if err := a.db.ORM.Callback().Create().Remove(callbackName); err != nil {
		t.Fatal(err)
	}
	requireCode(t, r, 500, 1500)
	after, err := os.ReadDir(a.cfg.File.LocalDir)
	if err != nil || len(after) != len(before) {
		t.Fatal("数据库失败遗留孤儿文件")
	}
	requireCode(t, callAPI(t, a, "DELETE", "/api/sys/files/"+other, nil, member.AccessToken), 200, 0)
	requireCode(t, replyOf(t, binaryAPI(t, a, "GET", "/api/sys/files/"+other+"/download", member.AccessToken)), 404, 1001)
	if err := a.db.ORM.Table("sys_file_cleanup").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("成功物理删除后补偿记录未移除")
	}
}

type failCacheDelete struct {
	remaining atomic.Int64
	calls     atomic.Int64
}

func (h *failCacheDelete) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h *failCacheDelete) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *failCacheDelete) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "del" {
			args := cmd.Args()
			if len(args) > 1 && strings.Contains(fmt.Sprint(args[1]), ":cache:") {
				h.calls.Add(1)
				if h.remaining.Add(-1) >= 0 {
					return errors.New("模拟缓存失效失败")
				}
			}
		}
		return next(ctx, cmd)
	}
}
func TestM3CacheInvalidationRetriesAndRefreshRecovery(t *testing.T) {
	a := seedApp(t)
	admin := adminLogin(t, a)
	id := objectID(t, callAPI(t, a, "POST", "/api/sys/configs", map[string]string{"name": "标题", "key": "app.retry", "value": "旧值"}, admin.AccessToken))
	requireCode(t, callAPI(t, a, "GET", "/api/sys/configs/key/app.retry", nil, admin.AccessToken), 200, 0)
	hook := new(failCacheDelete)
	hook.remaining.Store(3)
	a.cache.AddHook(hook)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/configs/"+id, map[string]string{"name": "标题", "key": "app.retry", "value": "新值"}, admin.AccessToken), 200, 0)
	if hook.calls.Load() != 3 {
		t.Fatalf("失效未有界重试 %d", hook.calls.Load())
	}
	var value string
	if err := a.db.ORM.Table("sys_config").Select("value").Where("id = ?", id).Scan(&value).Error; err != nil || value != "新值" {
		t.Fatal("缓存故障不能回滚已提交数据库")
	}
	ttl := a.cache.TTL(t.Context(), a.cfg.Redis.KeyPrefix+"cache:config:app.retry").Val()
	if ttl <= 0 || ttl > 30*time.Minute {
		t.Fatal("陈旧缓存缺少 TTL 边界")
	}
	requireCode(t, callAPI(t, a, "POST", "/api/sys/configs/cache/refresh", nil, admin.AccessToken), 200, 0)
	r := callAPI(t, a, "GET", "/api/sys/configs/key/app.retry", nil, admin.AccessToken)
	requireCode(t, r, 200, 0)
	if !strings.Contains(string(r.Data), "新值") {
		t.Fatal("refresh 未恢复缓存")
	}
}
func TestM3DuplicateSubmitLockOwnerAndTTL(t *testing.T) {
	a := seedApp(t)
	admin := adminLogin(t, a)
	role := objectID(t, callAPI(t, a, "POST", "/api/sys/roles", map[string]string{"name": "成员", "code": "lock_member"}, admin.AccessToken))
	uid, _ := createMember(t, a, admin, "lock_user", "", role)
	sum := sha256.Sum256([]byte("1:PUT:/api/sys/users/:id/password:" + uid))
	key := a.cfg.Redis.KeyPrefix + "lock:submit:" + hex.EncodeToString(sum[:])
	if err := a.cache.Set(t.Context(), key, "foreign-owner", time.Second).Err(); err != nil {
		t.Fatal(err)
	}
	path := "/api/sys/users/" + uid + "/password"
	requireCode(t, callAPI(t, a, "PUT", path, map[string]string{"password": "Reset-Password-789"}, admin.AccessToken), 409, 1002)
	if a.cache.Get(t.Context(), key).Val() != "foreign-owner" {
		t.Fatal("未持锁请求释放了其他所有者的锁")
	}
	if ttl := a.cache.PTTL(t.Context(), key).Val(); ttl <= 0 || ttl > time.Second {
		t.Fatal("关键写锁缺少 TTL")
	}
	if err := a.cache.Del(t.Context(), key).Err(); err != nil {
		t.Fatal(err)
	}
	requireCode(t, callAPI(t, a, "PUT", path, map[string]string{"password": "Reset-Password-789"}, admin.AccessToken), 200, 0)
	if a.cache.Exists(t.Context(), key).Val() != 0 {
		t.Fatal("完成后没有按所有者释放锁")
	}
}
