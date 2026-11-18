package file

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type idSource interface{ Next() (int64, error) }

// Service 管理校验、同所有者去重和持久删除补偿。
type Service struct {
	repo    repository
	storage Storage
	ids     idSource
	cfg     config.File
	log     *zap.Logger
	mu      sync.Mutex
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New 启动受生命周期管理的清理 worker。
func New(db *gorm.DB, storage Storage, ids idSource, cfg config.File, log *zap.Logger) *Service {
	s := &Service{repo: repository{db}, storage: storage, ids: ids, cfg: cfg, log: log}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Go(func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.flush()
			}
		}
	})
	return s
}
func view(r file) View {
	return View{r.ID, r.OriginalName, r.Size, r.Mime, "/api/sys/files/" + strconv.FormatInt(int64(r.ID), 10) + "/download", r.CreatedAt}
}

// Upload 校验双白名单，预登记补偿并在提交文件元数据时取消补偿。
func (s *Service) Upload(ctx context.Context, p security.Principal, name string, r io.Reader) (View, error) {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "." || name == "" || len([]rune(name)) > 255 || strings.ContainsAny(name, "\r\n\x00") {
		return View{}, response.ErrInvalidParam
	}
	body, err := io.ReadAll(io.LimitReader(r, s.cfg.MaxSize+1))
	if err != nil {
		return View{}, response.ErrInvalidParam
	}
	if len(body) == 0 || int64(len(body)) > s.cfg.MaxSize {
		return View{}, response.ErrInvalidParam
	}
	ext := strings.ToLower(filepath.Ext(name))
	mime := http.DetectContentType(body)
	mime, _, _ = strings.Cut(mime, ";")
	if !slices.Contains(s.cfg.Extensions, ext) || !slices.Contains(s.cfg.MIMETypes, mime) || !matchingType(ext, mime) {
		return View{}, response.ErrInvalidParam
	}
	sum := sha256.Sum256(body)
	digest := hex.EncodeToString(sum[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	old, err := s.repo.hash(ctx, p.ID, digest)
	if err == nil {
		return view(old), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return View{}, err
	}
	n, err := s.ids.Next()
	if err != nil {
		return View{}, err
	}
	key := strconv.FormatInt(n, 10) + ext
	task := cleanup{ID: model.ID(n), Path: key, Storage: "local", NextAttempt: time.Now().UTC()}
	if err := s.repo.queue(ctx, task); err != nil {
		return View{}, err
	}
	path, err := s.storage.Save(ctx, key, bytes.NewReader(body))
	if err != nil {
		return View{}, err
	}
	row := file{ID: model.ID(n), CreatedBy: p.ID, UpdatedBy: p.ID, Name: key, OriginalName: name, Path: path, Size: int64(len(body)), Mime: mime, Storage: "local", Hash: digest}
	if err := s.repo.create(ctx, &row, task.ID); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		s.clean(cleanupCtx, task.ID)
		cancel()
		if errors.Is(httpx.DBError(err), response.ErrConflict) {
			old, e := s.repo.hash(ctx, p.ID, digest)
			if e == nil {
				return view(old), nil
			}
		}
		return View{}, httpx.DBError(err)
	}
	return view(row), nil
}
func matchingType(ext, mime string) bool {
	switch ext {
	case ".txt":
		return mime == "text/plain"
	case ".png":
		return mime == "image/png"
	case ".jpg", ".jpeg":
		return mime == "image/jpeg"
	case ".pdf":
		return mime == "application/pdf"
	default:
		return false
	}
}

// Download 先检查所有者，再打开存储流。
func (s *Service) Download(ctx context.Context, p security.Principal, id model.ID) (Download, error) {
	row, err := s.repo.get(ctx, nil, id)
	if err != nil {
		return Download{}, httpx.DBError(err)
	}
	if !p.Admin() && row.CreatedBy != p.ID {
		return Download{}, response.ErrPermission
	}
	body, err := s.storage.Open(ctx, row.Path)
	if err != nil {
		return Download{}, response.ErrNotFound
	}
	return Download{view(row), body}, nil
}

// Remove 将元数据删除与物理补偿入队放在同一事务内。
func (s *Service) Remove(ctx context.Context, p security.Principal, id model.ID) error {
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := s.repo.get(ctx, tx, id)
		if err != nil {
			return httpx.DBError(err)
		}
		if !p.Admin() && row.CreatedBy != p.ID {
			return response.ErrPermission
		}
		n, err := s.ids.Next()
		if err != nil {
			return err
		}
		task := cleanup{ID: model.ID(n), Path: row.Path, Storage: row.Storage, NextAttempt: time.Now().UTC()}
		return s.repo.remove(ctx, tx, row, task)
	})
	if err == nil {
		s.flush()
	}
	return err
}
func (s *Service) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := s.repo.pending(ctx)
	if err != nil {
		s.log.Warn("文件补偿队列读取失败")
		return
	}
	for _, row := range rows {
		s.clean(ctx, row.ID)
	}
}

func (s *Service) clean(ctx context.Context, id model.ID) {
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 上传提交会在同一事务中删除此记录，行锁等待事务结果明确。
		row, err := s.repo.lockCleanup(ctx, tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if row.Storage != "local" {
			s.log.Error("文件补偿存储驱动不可用")
			return s.repo.retry(ctx, tx, row)
		}
		active, err := s.repo.activePath(ctx, tx, row.Storage, row.Path)
		if err != nil {
			return err
		}
		if !active {
			if err := s.storage.Delete(ctx, row.Path); err != nil {
				s.log.Warn("文件物理删除失败，继续等待补偿")
				return s.repo.retry(ctx, tx, row)
			}
		}
		return s.repo.done(ctx, tx, row.ID)
	})
	if err != nil {
		s.log.Warn("文件补偿未完成，后续重试")
	}
}

// Close 停止后台清理，在数据库关闭前完成最后一轮补偿。
func (s *Service) Close() error {
	s.cancel()
	s.wg.Wait()
	s.flush()
	if c, ok := s.storage.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
