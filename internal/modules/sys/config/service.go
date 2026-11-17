// Package config 提供参数管理、普通缓存和认证安全参数直接读取。
package config

import (
	"context"
	"errors"
	"strconv"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/cache"
	"gorm.io/gorm"
)

// Settings 的类型与边界在读取时检查，非法参数不能降低认证约束。
type Settings struct {
	CaptchaEnabled     bool
	MultiLogin         bool
	MaxSessions        int
	PasswordRetryLimit int
	LockMinutes        int
}

// Service 管理业务参数，认证读取不使用普通缓存。
type Service struct {
	db    *gorm.DB
	repo  repository
	cache *cache.Store
	ids   idSource
}

// New 创建参数业务，缓存与 ID 由装配层配置。
func New(db *gorm.DB) *Service { return &Service{db: db, repo: repository{db}} }

// Read 缺失键采用安全默认值，已有非法值则拒绝认证。
func (s *Service) Read(ctx context.Context) (Settings, error) {
	out := Settings{true, true, 5, 5, 10}
	var rows []parameter
	keys := []string{"sys.account.captchaEnabled", "sys.account.multiLogin", "sys.account.maxSessions", "sys.account.passwordRetryLimit", "sys.account.lockMinutes"}
	var err error
	rows, err = s.repo.keys(ctx, keys)
	if err != nil {
		return Settings{}, err
	}
	for _, row := range rows {
		switch row.Key {
		case keys[0], keys[1]:
			if row.Value != "true" && row.Value != "false" {
				return Settings{}, errors.New("认证布尔参数无效")
			}
			v, _ := strconv.ParseBool(row.Value)
			if row.Key == keys[0] {
				out.CaptchaEnabled = v
			} else {
				out.MultiLogin = v
			}
		default:
			n, err := strconv.Atoi(row.Value)
			if err != nil || n < 1 || n > 100 {
				return Settings{}, errors.New("认证整数参数无效")
			}
			switch row.Key {
			case keys[2]:
				out.MaxSessions = n
			case keys[3]:
				out.PasswordRetryLimit = n
			case keys[4]:
				out.LockMinutes = n
			}
		}
	}
	return out, nil
}
