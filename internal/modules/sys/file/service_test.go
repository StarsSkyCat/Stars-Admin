package file

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
)

func TestLocalStorageRejectsTraversalSymlinkAndOverwrite(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = storage.(*localStorage).Close() }()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../outside.txt", outside, "a/../b", "a\\b", ".", "link.txt"} {
		if body, err := storage.Open(t.Context(), path); err == nil {
			_ = body.Close()
			t.Fatalf("路径越界未拒绝 %s", path)
		}
	}
	if _, err := storage.Save(t.Context(), "safe.txt", strings.NewReader("one")); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.Save(t.Context(), "safe.txt", strings.NewReader("two")); err == nil {
		t.Fatal("覆盖已有文件")
	}
	body, err := storage.Open(t.Context(), "safe.txt")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil || string(raw) != "one" {
		t.Fatal("文件内容被覆盖")
	}
}
func TestUploadRejectsSizeAndMIMEMismatchBeforePersistence(t *testing.T) {
	s := &Service{cfg: config.File{MaxSize: 10, Extensions: []string{".txt", ".png"}, MIMETypes: []string{"text/plain", "image/png"}}}
	for _, in := range []struct{ name, body string }{{"x.txt", "01234567890"}, {"fake.png", "hello"}, {"x.exe", "hello"}, {"empty.txt", ""}} {
		if _, err := s.Upload(t.Context(), security.Principal{ID: 2}, in.name, strings.NewReader(in.body)); !errors.Is(err, response.ErrInvalidParam) {
			t.Fatalf("应在持久化前拒绝 %s", in.name)
		}
	}
}
