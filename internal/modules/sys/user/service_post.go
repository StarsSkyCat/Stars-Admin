package user

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

// Posts 校验有效岗位，关联写入仍在用户事务内。
type Posts interface {
	Validate(context.Context, *gorm.DB, []model.ID) error
}

// SetPosts 接入岗位引用能力。
func (s *Service) SetPosts(p Posts) { s.posts = p }

// GrantPosts 替换用户岗位，解绑后可以重新绑定。
func (s *Service) GrantPosts(ctx context.Context, p security.Principal, id model.ID, ids []model.ID) error {
	if err := httpx.UniqueIDs(ids); err != nil {
		return err
	}
	if err := s.gate.Allow(p, "sys:user:grant"); err != nil {
		return err
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		scoped, err := s.scopeContext(ctx, tx, p, "sys:user:grant")
		if err != nil {
			return nil, err
		}
		if _, err := s.repo.get(scoped, tx, id); err != nil {
			return nil, httpx.DBError(err)
		}
		if err := s.posts.Validate(ctx, tx, ids); err != nil {
			return nil, err
		}
		rows := []userPost{}
		for _, postID := range ids {
			n, err := s.ids.Next()
			if err != nil {
				return nil, err
			}
			rows = append(rows, userPost{ID: model.ID(n), CreatedBy: p.ID, UpdatedBy: p.ID, UserID: id, PostID: postID})
		}
		return nil, s.repo.replacePosts(ctx, tx, id, rows)
	})
}

// HasPosts 提供岗位删除保护。
func (s *Service) HasPosts(ctx context.Context, tx *gorm.DB, ids []model.ID) (bool, error) {
	return s.repo.hasPosts(ctx, tx, ids)
}

// RevokeAll 在持久事务提升版本，Redis 故障与应用重启后仍拒绝旧会话。
func (s *Service) RevokeAll(ctx context.Context, p security.Principal, id model.ID) error {
	if !p.Admin() {
		return response.ErrPermission
	}
	if err := s.gate.Allow(p, "sys:audit:forceLogout"); err != nil {
		return err
	}
	if id == 1 && !p.Admin() {
		return response.ErrPermission
	}
	return s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		if _, err := s.repo.get(ctx, tx, id); err != nil {
			return nil, httpx.DBError(err)
		}
		return s.repo.bump(ctx, tx, []model.ID{id})
	})
}
