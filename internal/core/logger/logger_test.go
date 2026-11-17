package logger

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"go.uber.org/zap"
)

func TestJSONFileAndLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "app.log")
	log, closeLog, err := New(config.Log{Level: "info", Format: "json", File: path, MaxSize: 1, MaxBackups: 1, MaxAge: 1})
	if err != nil {
		t.Fatal(err)
	}
	log.Debug("不应输出")
	log.Info("测试事件", zap.String("traceId", "test-id"))
	if err := closeLog(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entry map[string]any
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatal(err)
	}
	if entry["msg"] != "测试事件" || entry["traceId"] != "test-id" || entry["level"] != "info" {
		t.Fatal("结构化日志或级别过滤错误")
	}
	if _, _, err := New(config.Log{Level: "invalid"}); err == nil {
		t.Fatal("非法日志级别应被拒绝")
	}
}
