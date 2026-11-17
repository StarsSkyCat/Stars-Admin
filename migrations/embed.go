// Package migrations 内嵌版本迁移，使 CLI 不依赖工作目录中的 SQL 文件。
package migrations

import "embed"

// Files 包含正式的升降级脚本。
//
//go:embed *.sql
var Files embed.FS
