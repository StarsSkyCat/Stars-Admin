//go:build integration

package log

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/scheduler"
	"github.com/StarsSkyCat/Stars-Admin/internal/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
)

func TestJobLogRetriesCountsLossAndDrainsBeforeDatabaseClose(t *testing.T) {
	cfg, err := config.Load("../../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	if err := deps.DB.Exec("TRUNCATE job_log").Error; err != nil {
		t.Fatal(err)
	}
	var attempts atomic.Int64
	name := "m4:job-log-failure"
	if err := deps.DB.Callback().Create().Before("gorm:create").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == "job_log" {
			attempts.Add(1)
			tx.AddError(errors.New("模拟日志持久化故障"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	logger, recorded := observer.New(zap.WarnLevel)
	service := New(deps.DB, deps.IDs, zap.New(logger))
	service.Submit(scheduler.Event{TaskID: 700, Name: "失败日志", Target: "safe", Params: "{}", Status: 2, StartedAt: time.Now().UTC()})
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	if err := deps.DB.Callback().Create().Remove(name); err != nil {
		t.Fatal(err)
	}
	if attempts.Load() != 3 || service.Dropped() != 1 || recorded.FilterMessage("任务日志重试耗尽").Len() != 1 {
		t.Fatal("任务日志失败缺少有界重试、告警或计数")
	}
	next := New(deps.DB, deps.IDs, deps.Logger)
	for range 75 {
		next.Submit(scheduler.Event{TaskID: 701, Name: "成功日志", Target: "safe", Params: "{}", Status: 1, StartedAt: time.Now().UTC()})
	}
	if err := next.Close(); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := deps.DB.Model(&entry{}).Count(&count).Error; err != nil || count != 75 {
		t.Fatalf("关闭前未排空日志 %d %v", count, err)
	}
}
