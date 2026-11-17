package audit

import (
	"context"
	"strconv"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/session"
	"github.com/StarsSkyCat/Stars-Admin/pkg/excel"
)

// Sessions 提供已核对临时会话状态的在线摘要。
type Sessions interface {
	OnlineList(context.Context) ([]session.Online, error)
}

// Users 提供持久版本核对与全部会话撤销。
type Users interface {
	Account(context.Context, model.ID) (security.Account, error)
	RevokeAll(context.Context, security.Principal, model.ID) error
}

// Configure 接入会话摘要与持久撤销能力。
func (s *Service) Configure(sessions Sessions, users Users) { s.sessions = sessions; s.users = users }
func normalize(q *Query) error {
	if err := q.Normalize(); err != nil {
		return err
	}
	if q.Status < 0 || q.Status > 2 || len(q.Username) > 64 || (q.OrderBy != "" && q.OrderBy != "id desc") {
		return response.ErrInvalidParam
	}
	return nil
}

// Logins 返回登录日志分页。
func (s *Service) Logins(ctx context.Context, q Query) (response.Page[LoginView], error) {
	if err := normalize(&q); err != nil {
		return response.Page[LoginView]{}, err
	}
	_, rows, total, err := s.repo.list(ctx, q, false)
	out := response.Page[LoginView]{List: []LoginView{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, r := range rows {
		out.List = append(out.List, LoginView{r.ID, r.Username, r.IP, r.Status, r.Msg, r.LoginAt})
	}
	return out, err
}

// Operations 返回操作日志分页。
func (s *Service) Operations(ctx context.Context, q Query) (response.Page[OperView], error) {
	if err := normalize(&q); err != nil {
		return response.Page[OperView]{}, err
	}
	rows, _, total, err := s.repo.list(ctx, q, true)
	out := response.Page[OperView]{List: []OperView{}, Total: total, Page: q.Page, PageSize: q.PageSize}
	for _, r := range rows {
		out.List = append(out.List, OperView{r.ID, r.UserID, r.Username, r.Module, r.Action, r.TraceID, r.Method, r.URL, r.IP, r.Response, r.Status, r.CostMS, r.OperAt})
	}
	return out, err
}

// Clean 仅由独立危险权限入口调用。
func (s *Service) Clean(ctx context.Context, oper bool) error { return s.repo.clean(ctx, oper) }

// Online 同时核对数据库当前状态和会话版本。
func (s *Service) Online(ctx context.Context, q response.PageQuery) (response.Page[OnlineView], error) {
	if err := q.Normalize(); err != nil {
		return response.Page[OnlineView]{}, err
	}
	rows, err := s.sessions.OnlineList(ctx)
	if err != nil {
		return response.Page[OnlineView]{}, response.ErrUnavailable
	}
	out := response.Page[OnlineView]{List: []OnlineView{}, Page: q.Page, PageSize: q.PageSize}
	for _, r := range rows {
		account, e := s.users.Account(ctx, r.UserID)
		if e != nil {
			continue
		}
		if account.Status == 1 && len(account.RoleIDs) > 0 && r.Version == account.AuthVersion {
			out.List = append(out.List, OnlineView(r))
		}
	}
	out.Total = int64(len(out.List))
	start := min((q.Page-1)*q.PageSize, len(out.List))
	out.List = out.List[start:min(start+q.PageSize, len(out.List))]
	return out, nil
}

// ForceLogout 委托持久撤销，所有旧 Access 与 Refresh 一起失效。
func (s *Service) ForceLogout(ctx context.Context, p security.Principal, id model.ID) error {
	return s.users.RevokeAll(ctx, p, id)
}

// Export 返回不含认证材料且已防公式注入的 Excel。
func (s *Service) Export(ctx context.Context, q Query, oper bool) ([]byte, error) {
	if err := normalize(&q); err != nil {
		return nil, err
	}
	q.export = true
	ops, logins, _, err := s.repo.list(ctx, q, oper)
	if err != nil {
		return nil, err
	}
	rows := [][]string{}
	if oper {
		rows = append(rows, []string{"id", "username", "method", "url", "status", "operAt"})
		for _, r := range ops {
			rows = append(rows, []string{strconv.FormatInt(int64(r.ID), 10), r.Username, r.Method, r.URL, strconv.Itoa(r.Status), r.OperAt.Format("2006-01-02T15:04:05Z")})
		}
	} else {
		rows = append(rows, []string{"id", "username", "ip", "status", "msg", "loginAt"})
		for _, r := range logins {
			rows = append(rows, []string{strconv.FormatInt(int64(r.ID), 10), r.Username, r.IP, strconv.Itoa(r.Status), r.Msg, r.LoginAt.Format("2006-01-02T15:04:05Z")})
		}
	}
	return excel.Write(rows)
}
