// Package core 定义模块装配所需的框架依赖，不引用业务模块。
package core

import (
	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/pkg/snowflake"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Deps 仅用于模块构造，具体功能仍应接收所需的小接口。
type Deps struct {
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Logger *zap.Logger
	IDs    *snowflake.Generator
}
