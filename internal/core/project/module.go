// Package project 读取构建时模块信息或实际 go.mod，不硬编码项目 import 根路径。
package project

import (
	"errors"
	"os"
	"path/filepath"
	"runtime/debug"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// ModulePath 优先读取当前实际 go.mod，精简生产镜像回退到二进制构建信息。
func ModulePath() (string, error) {
	dir, err := os.Getwd()
	if err == nil {
		for {
			data, e := os.ReadFile(filepath.Join(dir, "go.mod"))
			if e == nil {
				path := modfile.ModulePath(data)
				if err := module.CheckPath(path); err != nil {
					return "", err
				}
				return path, nil
			}
			next := filepath.Dir(dir)
			if next == dir {
				break
			}
			dir = next
		}
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Path != "" {
		if err := module.CheckPath(info.Main.Path); err == nil {
			return info.Main.Path, nil
		}
	}
	return "", errors.New("无法确定实际 Go 模块路径")
}
