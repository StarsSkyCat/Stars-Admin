// Package safeoutput 创建受根目录约束且不覆盖的文本文件集合或 zip。
package safeoutput

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// File 是应用生成的相对路径与内容。
type File struct {
	Path    string
	Content string
}

func valid(name string) bool {
	if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\:\x00") {
		return false
	}
	for part := range strings.SplitSeq(name, "/") {
		if slices.Contains([]string{"AGENTS.md", "go.mod", "go.sum", "module.go", ".git", ".codex", "configs", "config.yaml"}, part) {
			return false
		}
	}
	return true
}
func validate(files []File) error {
	if len(files) == 0 || len(files) > 100 {
		return errors.New("输出文件数量无效")
	}
	seen := map[string]bool{}
	size := 0
	for _, f := range files {
		size += len(f.Content)
		if !valid(f.Path) || seen[f.Path] || size > 10*1024*1024 {
			return errors.New("输出路径或大小无效")
		}
		seen[f.Path] = true
	}
	return nil
}

// ZIP 所有条目先校验为安全相对路径。
func ZIP(files []File) ([]byte, error) {
	if err := validate(files); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, f := range files {
		dst, err := w.Create(f.Path)
		if err != nil {
			return nil, err
		}
		if _, err = dst.Write([]byte(f.Content)); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// Local 完整预检后以 O_EXCL 写入，任一失败清理本轮创建的文件与空目录。
func Local(rootDir string, files []File) (err error) {
	if err := validate(files); err != nil {
		return err
	}
	absolute, err := filepath.Abs(rootDir)
	if err != nil {
		return err
	}
	for current := absolute; ; current = filepath.Dir(current) {
		info, e := os.Lstat(current)
		if e == nil && info.Mode()&os.ModeSymlink != 0 {
			return os.ErrPermission
		}
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	if err := os.MkdirAll(absolute, 0700); err != nil {
		return err
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	for _, f := range files {
		parts := strings.Split(f.Path, "/")
		for i := 1; i <= len(parts); i++ {
			p := strings.Join(parts[:i], "/")
			info, e := root.Lstat(p)
			if e == nil {
				if info.Mode()&os.ModeSymlink != 0 || i == len(parts) || !info.IsDir() {
					return os.ErrExist
				}
			} else if !errors.Is(e, os.ErrNotExist) {
				return e
			}
		}
	}
	written := []string{}
	directories := []string{}
	defer func() {
		if err != nil {
			for _, p := range written {
				_ = root.Remove(p)
			}
			slices.Reverse(directories)
			for _, p := range directories {
				_ = root.Remove(p)
			}
		}
	}()
	for _, f := range files {
		dir := path.Dir(f.Path)
		if dir != "." {
			parts := strings.Split(dir, "/")
			for i := 1; i <= len(parts); i++ {
				p := strings.Join(parts[:i], "/")
				if e := root.Mkdir(p, 0700); e == nil {
					directories = append(directories, p)
				} else if !errors.Is(e, os.ErrExist) {
					return e
				}
			}
		}
		dst, e := root.OpenFile(f.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		written = append(written, f.Path)
		_, e = dst.Write([]byte(f.Content))
		if closeErr := dst.Close(); e == nil {
			e = closeErr
		}
		if e != nil {
			return e
		}
	}
	return nil
}
