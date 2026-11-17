package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoverageUnionThresholdAndMissingSource(t *testing.T) {
	dir := t.TempDir()
	write := func(name, data string) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write("go.mod", "module example.com/test\n\ngo 1.27.0\n")
	write("internal/modules/demo/item/service.go", "package item\n")
	// 非 service 文件和测试文件不能抬高或降低业务统计。
	write("internal/modules/demo/item/service_test.go", "package item\n")
	a := write("unit.out", "mode: atomic\nexample.com/test/internal/modules/demo/item/service.go:1.1,2.1 3 1\nexample.com/test/internal/modules/demo/item/service.go:3.1,4.1 2 0\nexample.com/test/internal/modules/demo/item/handler.go:1.1,9.1 900 1\n")
	b := write("integration.out", "mode: atomic\nexample.com/test/internal/modules/demo/item/service.go:1.1,2.1 3 1\nexample.com/test/internal/modules/demo/item/service.go:3.1,4.1 2 1\n")
	out, report := filepath.Join(dir, "merged.out"), filepath.Join(dir, "report.md")
	if err := run(dir, []string{a, b}, out, report, 100); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(report)
	if err != nil || !strings.Contains(string(raw), "5/5 = 100.00%") {
		t.Fatal("未按语句块取并集")
	}
	if err := run(dir, []string{a, a}, out, report, 60.01); err == nil {
		t.Fatal("低于门槛仍通过")
	}
	write("internal/modules/demo/item/service_export.go", "package item\n")
	if err := run(dir, []string{a, b}, out, report, 60); err == nil {
		t.Fatal("漏采 service 仍通过")
	}
	bad := write("bad.out", "mode: atomic\nexample.com/test/internal/modules/demo/item/service.go:1.1,2.1 4 1\n")
	if err := run(dir, []string{a, bad}, out, report, 60); err == nil {
		t.Fatal("不同源码块仍被合并")
	}
}
