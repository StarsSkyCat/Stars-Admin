// Package configs 嵌入与生产镜像一致的 Casbin 模型。
package configs

import _ "embed"

// RBACModel 是唯一的 Casbin 模型来源，不依赖工作目录。
//
//go:embed casbin_model.conf
var RBACModel string
