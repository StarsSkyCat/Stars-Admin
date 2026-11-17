// Package file 提供受保护文件管理，不公开存储路径。
package file

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Storage 是文件存储驱动的能力边界。
type Storage interface {
	Save(context.Context, string, io.Reader) (string, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}
type localStorage struct{ root *os.Root }

// NewLocal 打开本地根目录，os.Root 拒绝符号链接逃逸和目录穿越。
func NewLocal(dir string) (Storage, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &localStorage{root}, nil
}
func validPath(name string) bool {
	return name != "" && filepath.IsLocal(name) && filepath.Base(name) == name && !strings.ContainsAny(name, "/\\:") && name != "."
}
func (s *localStorage) Save(ctx context.Context, name string, r io.Reader) (string, error) {
	if !validPath(name) {
		return "", os.ErrPermission
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	f, err := s.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(f, r)
	err = errors.Join(copyErr, f.Close(), ctx.Err())
	if err != nil {
		_ = s.root.Remove(name)
		return "", err
	}
	return name, nil
}
func (s *localStorage) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	if !validPath(path) {
		return nil, os.ErrPermission
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.root.Open(path)
}
func (s *localStorage) Delete(ctx context.Context, path string) error {
	if !validPath(path) {
		return os.ErrPermission
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	err := s.root.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
func (s *localStorage) Close() error { return s.root.Close() }
