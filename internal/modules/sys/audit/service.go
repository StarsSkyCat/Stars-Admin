// Package audit 提供认证阶段的最小登录日志写入能力。
package audit

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"gorm.io/gorm"
)

type loginLog struct {
	model.LogBase
	Username string
	IP       string
	Status   int
	Msg      string
	LoginAt  time.Time
}

func (loginLog) TableName() string { return "sys_login_log" }

type idSource interface{ Next() (int64, error) }

// Service 使用独立的事件字段白名单，不采集原始请求或令牌。
type Service struct {
	db  *gorm.DB
	ids idSource
}

// New 创建登录日志业务。
func New(db *gorm.DB, ids idSource) *Service { return &Service{db, ids} }

// Login 只接受服务生成的结果文案，不接收客户端密码和令牌。
func (s *Service) Login(ctx context.Context, username, ip string, success bool, message string) error {
	id, err := s.ids.Next()
	if err != nil {
		return err
	}
	status := 2
	if success {
		status = 1
	}
	if !utf8.ValidString(username) {
		username = ""
	}
	if utf8.RuneCountInString(username) > 64 {
		username = string([]rune(username)[:64])
	}
	return s.db.WithContext(ctx).Create(&loginLog{ID: model.ID(id), Username: username, IP: ip, Status: status, Msg: message, LoginAt: time.Now().UTC()}).Error
}
