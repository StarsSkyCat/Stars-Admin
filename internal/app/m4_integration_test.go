//go:build integration

package app

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	joblog "github.com/StarsSkyCat/Stars-Admin/internal/modules/job/log"
	"github.com/StarsSkyCat/Stars-Admin/internal/modules/job/task"
	"go.uber.org/zap"
)

func TestM4JobRoutesExecutionLogsAndGlobalAdminBoundary(t *testing.T) {
	a := seedApp(t)
	initial := loginUser(t, a, "admin", firstPassword)
	requireCode(t, callAPI(t, a, "GET", "/api/job/tasks", nil, initial.AccessToken), 403, 2008)
	requireCode(t, callAPI(t, a, "PUT", "/api/sys/auth/password", map[string]string{"oldPassword": firstPassword, "newPassword": changedPassword}, initial.AccessToken), 200, 0)
	admin := loginUser(t, a, "admin", changedPassword)
	requireCode(t, callAPI(t, a, "GET", "/api/job/tasks", nil, ""), 401, 2000)
	role := objectID(t, callAPI(t, a, "POST", "/api/sys/roles", map[string]string{"name": "工具读者", "code": "tools_reader"}, admin.AccessToken))
	grantPermissions(t, a, admin, role, "job:task:list", "job:task:run", "job:log:list", "gen:table:list", "gen:table:gen")
	_, member := createMember(t, a, admin, "tools_reader", "", role)
	for _, path := range []string{"/api/job/tasks", "/api/job/logs", "/api/gen/tables", "/api/gen/tables/db"} {
		requireCode(t, callAPI(t, a, "GET", path, nil, member.AccessToken), 403, 2004)
	}
	in := map[string]any{"name": "验收任务", "cron": "0 0 0 1 1 *", "target": "demoTask", "params": `{"message":"private-message","delayMs":300}`}
	for _, bad := range []map[string]any{{"name": "错误目标", "cron": "0 0 0 1 1 *", "target": "sh"}, {"name": "错误Cron", "cron": "* * * * *", "target": "demoTask"}, {"name": "非法参数", "cron": "* * * * * *", "target": "demoTask", "params": `{"password":"hidden"}`}, {"name": "绕过暂停", "cron": "* * * * * *", "target": "demoTask", "status": 1}} {
		requireCode(t, callAPI(t, a, "POST", "/api/job/tasks", bad, admin.AccessToken), 400, 1000)
	}
	created := callAPI(t, a, "POST", "/api/job/tasks", in, admin.AccessToken)
	id := objectID(t, created)
	if created.As[task.View](t).Status != 2 {
		t.Fatal("新任务未默认暂停")
	}
	requireCode(t, callAPI(t, a, "GET", "/api/job/tasks/cron/next?expr="+url.QueryEscape("0 */5 * * * *"), nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "PUT", "/api/job/tasks/"+id+"/status", map[string]int{"status": 1}, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "PUT", "/api/job/tasks/"+id, in, admin.AccessToken), 200, 0)
	first := callAPI(t, a, "POST", "/api/job/tasks/"+id+"/run", nil, admin.AccessToken)
	requireCode(t, first, 200, 0)
	if first.As[task.RunResult](t).State != "queued" {
		t.Fatal("未投递手动执行")
	}
	second := callAPI(t, a, "POST", "/api/job/tasks/"+id+"/run", nil, admin.AccessToken)
	requireCode(t, second, 200, 0)
	if second.As[task.RunResult](t).State != "skipped" {
		t.Fatal("手动入口未共享门禁")
	}
	deadline := time.Now().Add(3 * time.Second)
	var logs response.Page[joblog.View]
	for time.Now().Before(deadline) {
		r := callAPI(t, a, "GET", "/api/job/logs?taskId="+id, nil, admin.AccessToken)
		requireCode(t, r, 200, 0)
		logs = r.As[response.Page[joblog.View]](t)
		if logs.Total >= 2 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if logs.Total != 2 {
		t.Fatal("缺少真实执行或跳过日志")
	}
	success, skipped := false, false
	for _, entry := range logs.List {
		if strings.Contains(entry.Params+entry.ErrorMsg, "private-message") {
			t.Fatal("日志泄漏任务参数")
		}
		if entry.Status == 1 {
			success = true
		}
		if entry.Status == 2 && strings.Contains(entry.ErrorMsg, "跳过") {
			skipped = true
		}
	}
	if !success || !skipped {
		t.Fatal("执行与跳过结果混淆")
	}
	requireCode(t, callAPI(t, a, "DELETE", "/api/job/tasks/"+id+",9223372036854775807", nil, admin.AccessToken), 404, 1001)
	requireCode(t, callAPI(t, a, "GET", "/api/job/tasks/"+id, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "PUT", "/api/job/tasks/"+id+"/status", map[string]int{"status": 2}, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "DELETE", "/api/job/tasks/"+id, nil, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, a, "DELETE", "/api/job/logs/clean", nil, admin.AccessToken), 200, 0)
}
func TestM4StoredInvalidTaskIsNotReadyAndCanBeRepaired(t *testing.T) {
	a := seedApp(t)
	adminLogin(t, a)
	if err := a.db.ORM.Exec("INSERT INTO job_task(id,created_at,updated_at,name,cron,target,params,status) VALUES(900001,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'错误配置','* * * * * *','unknown','{}',1)").Error; err != nil {
		t.Fatal(err)
	}
	cfg := a.cfg
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err := New(t.Context(), cfg, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := restored.Close(); err != nil {
			t.Error(err)
		}
	})
	requireCode(t, callAPI(t, restored, "GET", "/api/health/ready", nil, ""), 503, 1501)
	admin := loginUser(t, restored, "admin", changedPassword)
	requireCode(t, callAPI(t, restored, "PUT", "/api/job/tasks/900001", map[string]any{"name": "修复配置", "cron": "0 0 0 1 1 *", "target": "demoTask", "params": "{}"}, admin.AccessToken), 200, 0)
	requireCode(t, callAPI(t, restored, "GET", "/api/health/ready", nil, ""), 200, 0)
	requireCode(t, callAPI(t, restored, "DELETE", "/api/job/tasks/900001", nil, admin.AccessToken), 200, 0)
}
