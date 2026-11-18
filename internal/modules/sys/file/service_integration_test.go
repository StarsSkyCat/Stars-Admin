//go:build integration

package file

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/StarsSkyCat/Stars-Admin/internal/testutil"
)

type failingDelete struct {
	Storage
	fail atomic.Bool
}

func TestCommittedUploadSurvivesCompensation(t *testing.T) {
	cfg, err := config.Load("../../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	storage, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := New(deps.DB, storage, deps.IDs, cfg.File, deps.Logger)
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	p := security.Principal{ID: 777}
	view, err := s.Upload(t.Context(), p, "committed.txt", strings.NewReader("commit fixture"))
	if err != nil {
		t.Fatal(err)
	}
	// 模拟调用方遗失成功回执后尝试补偿已提交的上传。
	s.clean(t.Context(), view.ID)
	row, err := s.repo.get(t.Context(), nil, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := deps.IDs.Next()
	if err != nil {
		t.Fatal(err)
	}
	task := cleanup{ID: model.ID(taskID), Path: row.Path, Storage: row.Storage, NextAttempt: time.Now().UTC()}
	if err := s.repo.queue(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	s.clean(t.Context(), task.ID)
	out, err := s.Download(t.Context(), p, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(out.Body)
	if err := errors.Join(readErr, out.Body.Close()); err != nil || string(body) != "commit fixture" {
		t.Fatalf("有效上传被补偿清理：%v", err)
	}
}

func (s *failingDelete) Delete(ctx context.Context, path string) error {
	if s.fail.Load() {
		return errors.New("模拟物理删除失败")
	}
	return s.Storage.Delete(ctx, path)
}
func TestFileCompensationSurvivesServiceRestart(t *testing.T) {
	cfg, err := config.Load("../../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	cfg.File.LocalDir = t.TempDir()
	storage, err := NewLocal(cfg.File.LocalDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = storage.(io.Closer).Close() }()
	fault := &failingDelete{Storage: storage}
	fault.fail.Store(true)
	s := New(deps.DB, fault, deps.IDs, cfg.File, deps.Logger)
	p := security.Principal{ID: 777}
	out, err := s.Upload(t.Context(), p, "cleanup.txt", strings.NewReader("cleanup fixture"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Remove(t.Context(), p, out.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	var tasks []cleanup
	if err := deps.DB.Where("storage = ?", "local").Find(&tasks).Error; err != nil || len(tasks) != 1 {
		t.Fatalf("未持久保存待清理文件 %v %d", err, len(tasks))
	}
	if _, err := os.Stat(cfg.File.LocalDir + "/" + tasks[0].Path); err != nil {
		t.Fatal("物理文件应等待补偿")
	}
	fault.fail.Store(false)
	if err := deps.DB.Model(&cleanup{}).Where("id = ?", tasks[0].ID).Update("next_attempt", "2000-01-01T00:00:00Z").Error; err != nil {
		t.Fatal(err)
	}
	next := New(deps.DB, fault, deps.IDs, cfg.File, deps.Logger)
	next.flush()
	if err := next.Close(); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := deps.DB.Model(&cleanup{}).Where("id = ?", tasks[0].ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("重启后补偿未完成")
	}
	if _, err := os.Stat(cfg.File.LocalDir + "/" + tasks[0].Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("补偿未删除文件")
	}
}
