package safeoutput

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOutputRejectsUnsafePathsAndPreservesExistingFiles(t *testing.T) {
	for _, path := range []string{"../escape", "/absolute", "a\\b", "C:/x", "go.mod", "internal/modules/a/b/module.go", ".git/config"} {
		if _, err := ZIP([]File{{path, "bad"}}); err == nil {
			t.Fatalf("不安全zip路径被接受 %s", path)
		}
	}
	root := t.TempDir()
	files := []File{{"internal/modules/business/example/service.go", "original"}, {"migration-drafts/create_demo.up.sql", "-- draft"}}
	if err := Local(root, files); err != nil {
		t.Fatal(err)
	}
	if err := Local(root, files); !errors.Is(err, os.ErrExist) {
		t.Fatal("覆盖已有源码")
	}
	raw, err := os.ReadFile(filepath.Join(root, files[0].Path))
	if err != nil || string(raw) != "original" {
		t.Fatal("原文件被修改")
	}
	data, err := ZIP(files)
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(z.File) != 2 {
		t.Fatal("ZIP输出无效")
	}
}
func TestLocalRejectsSymlinkEscapeBeforeWriting(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "internal")); err != nil {
		t.Fatal(err)
	}
	if err := Local(root, []File{{"internal/modules/business/example/model.go", "bad"}}); err == nil {
		t.Fatal("允许符号链接逃逸")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatal("越界写入了物理文件")
	}
}
