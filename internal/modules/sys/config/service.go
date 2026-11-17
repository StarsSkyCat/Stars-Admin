// Package config 提供认证所需的最小只读业务参数，不开放管理路由。
package config

import (
	"context"
	"errors"
	"strconv"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"gorm.io/gorm"
)

type parameter struct {
	model.Base
	Key   string
	Value string
}

func (parameter) TableName() string { return "sys_config" }

// Settings 的类型与边界在读取时检查，非法参数不能降低认证约束。
type Settings struct {
	CaptchaEnabled     bool
	MultiLogin         bool
	MaxSessions        int
	PasswordRetryLimit int
	LockMinutes        int
}

// Service 只持有数据库只读能力，不提供通用配置更新。
type Service struct{ db *gorm.DB }

// New 创建只读配置业务。
func New(db *gorm.DB) *Service { return &Service{db} }

// Read 缺失键采用安全默认值，已有非法值则拒绝认证。
func (s *Service) Read(ctx context.Context) (Settings, error) {
	out := Settings{true, true, 5, 5, 10}
	var rows []parameter
	keys := []string{"sys.account.captchaEnabled", "sys.account.multiLogin", "sys.account.maxSessions", "sys.account.passwordRetryLimit", "sys.account.lockMinutes"}
	if err := s.db.WithContext(ctx).Where("key IN ?", keys).Find(&rows).Error; err != nil {
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
