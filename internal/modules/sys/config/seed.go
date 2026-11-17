package config

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type idSource interface{ Next() (int64, error) }

// Seed 仅补缺失参数，冲突时保留已有值与软删除状态。
func (s *Service) Seed(ctx context.Context, tx *gorm.DB, ids idSource) error {
	values := [][3]string{{"验证码开关", "sys.account.captchaEnabled", "true"}, {"多点登录", "sys.account.multiLogin", "true"}, {"会话数量上限", "sys.account.maxSessions", "5"}, {"密码重试次数", "sys.account.passwordRetryLimit", "5"}, {"账号锁定分钟", "sys.account.lockMinutes", "10"}}
	for _, value := range values {
		id, err := ids.Next()
		if err != nil {
			return err
		}
		row := struct {
			model.Base
			Name     string
			Key      string
			Value    string
			IsSystem bool
		}{ID: model.ID(id), Name: value[0], Key: value[1], Value: value[2], IsSystem: true}
		if err := tx.WithContext(ctx).Table("sys_config").Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
