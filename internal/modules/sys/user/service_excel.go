package user

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/StarsSkyCat/Stars-Admin/pkg/excel"
	"github.com/StarsSkyCat/Stars-Admin/pkg/hash"
	"gorm.io/gorm"
)

// ImportError 仅描述出错行，不回显密码或原始单元格内容。
type ImportError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// ImportResult 任一错误时 Imported 恒为零。
type ImportResult struct {
	Imported int           `json:"imported"`
	Errors   []ImportError `json:"errors"`
}
type importRow struct {
	input   CreateInput
	roles   []model.ID
	posts   []model.ID
	encoded string
}

var importHeaders = []string{"username", "nickname", "password", "deptId", "roleIds", "postIds", "email", "phone", "gender"}

func parseIDs(text string) ([]model.ID, error) {
	ids := []model.ID{}
	if strings.TrimSpace(text) == "" {
		return ids, nil
	}
	for part := range strings.SplitSeq(text, ",") {
		id, err := httpx.ParseID(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, httpx.UniqueIDs(ids)
}

// Import 校验全部行后在单一事务写入用户及角色、岗位关系。
func (s *Service) Import(ctx context.Context, p security.Principal, data []byte) (ImportResult, error) {
	result := ImportResult{Errors: []ImportError{}}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if len(data) > 5*1024*1024 {
		return result, response.ErrInvalidParam
	}
	cells, err := excel.Read(data, 1001)
	if err != nil {
		return result, response.ErrInvalidParam
	}
	for i, key := range importHeaders {
		if cells[0][i] != key {
			return result, response.ErrInvalidParam
		}
	}
	if len(cells) < 2 {
		return result, response.ErrInvalidParam
	}
	rows := []importRow{}
	seen := map[string]bool{}
	for i, cols := range cells[1:] {
		row := importRow{input: CreateInput{Username: strings.ToLower(strings.TrimSpace(cols[0])), Nickname: strings.TrimSpace(cols[1]), Password: cols[2]}}
		var rowErr error
		if !usernamePattern.MatchString(row.input.Username) || row.input.Username == "admin" || row.input.Nickname == "" || len([]rune(row.input.Nickname)) > 64 || seen[row.input.Username] {
			rowErr = response.ErrInvalidParam
		}
		seen[row.input.Username] = true
		if cols[3] != "" {
			id, e := httpx.ParseID(cols[3])
			if e != nil {
				rowErr = e
			} else {
				row.input.DeptID = new(id)
			}
		}
		if cols[6] != "" {
			row.input.Email = new(cols[6])
		}
		if cols[7] != "" {
			row.input.Phone = new(cols[7])
		}
		if len(cols[6]) > 128 || len(cols[7]) > 20 {
			rowErr = response.ErrInvalidParam
		}
		if cols[8] != "" {
			gender, e := strconv.Atoi(cols[8])
			if e != nil || gender < 0 || gender > 2 {
				rowErr = response.ErrInvalidParam
			}
			row.input.Gender = gender
		}
		row.roles, err = parseIDs(cols[4])
		if err != nil {
			rowErr = err
		}
		row.posts, err = parseIDs(cols[5])
		if err != nil {
			rowErr = err
		}
		row.encoded, err = hash.Password(row.input.Password, s.cost)
		if err != nil {
			rowErr = response.ErrInvalidParam
		}
		if rowErr != nil {
			result.Errors = append(result.Errors, ImportError{i + 2, "用户名、资料、初始密码或关联 ID 无效"})
		}
		rows = append(rows, row)
		if ctx.Err() != nil {
			return result, response.ErrInvalidParam
		}
	}
	if len(result.Errors) > 0 {
		return result, nil
	}
	validationFailed := errors.New("导入行校验失败")
	err = s.gate.Change(ctx, func(tx *gorm.DB) ([]security.Revocation, error) {
		scoped, e := s.access.Context(ctx, tx, p, "sys:user:import")
		if e != nil {
			return nil, e
		}
		if slices.ContainsFunc(rows, func(row importRow) bool {
			return len(row.roles) != 0 || len(row.posts) != 0
		}) {
			if e := s.gate.Allow(p, "sys:user:grant"); e != nil {
				return nil, e
			}
		}
		for i, row := range rows {
			var rowErr error
			if row.input.DeptID != nil {
				rowErr = s.checkDepartment(scoped, *row.input.DeptID)
				if rowErr == nil {
					rowErr = s.depts.Exists(ctx, tx, *row.input.DeptID)
				}
			} else {
				scope, ok := security.CurrentScope(scoped)
				if ok && !scope.All {
					rowErr = response.ErrPermission
				}
			}
			if rowErr == nil {
				rowErr = s.roles.CanAssign(ctx, tx, p, 0, row.roles)
			}
			if rowErr == nil {
				rowErr = s.posts.Validate(ctx, tx, row.posts)
			}
			exists, e := s.repo.nameExists(ctx, tx, row.input.Username)
			if e != nil {
				return nil, e
			}
			if exists {
				rowErr = response.ErrConflict
			}
			if rowErr != nil {
				if _, ok := errors.AsType[*response.BizError](rowErr); !ok {
					return nil, rowErr
				}
				result.Errors = append(result.Errors, ImportError{i + 2, "用户名已占用或部门、角色、岗位超出允许范围"})
			}
		}
		if len(result.Errors) > 0 {
			return nil, validationFailed
		}
		for _, row := range rows {
			n, e := s.ids.Next()
			if e != nil {
				return nil, e
			}
			u := user{ID: model.ID(n), CreatedBy: p.ID, UpdatedBy: p.ID, Username: row.input.Username, Nickname: row.input.Nickname, Password: row.encoded, DeptID: row.input.DeptID, Email: row.input.Email, Phone: row.input.Phone, Gender: row.input.Gender, Status: 1, AuthVersion: 1}
			if e = s.repo.save(ctx, tx, &u, true); e != nil {
				return nil, httpx.DBError(e)
			}
			roles := []userRole{}
			for _, id := range row.roles {
				n, e := s.ids.Next()
				if e != nil {
					return nil, e
				}
				roles = append(roles, userRole{ID: model.ID(n), CreatedBy: p.ID, UpdatedBy: p.ID, UserID: u.ID, RoleID: id})
			}
			if e = s.repo.replaceRoles(ctx, tx, u.ID, roles); e != nil {
				return nil, e
			}
			posts := []userPost{}
			for _, id := range row.posts {
				n, e := s.ids.Next()
				if e != nil {
					return nil, e
				}
				posts = append(posts, userPost{ID: model.ID(n), CreatedBy: p.ID, UpdatedBy: p.ID, UserID: u.ID, PostID: id})
			}
			if e = s.repo.replacePosts(ctx, tx, u.ID, posts); e != nil {
				return nil, e
			}
		}
		return nil, nil
	})
	if errors.Is(err, validationFailed) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Imported = len(rows)
	return result, nil
}

// Export 在同一次安全读取中授权并查询，释放读锁后编码表格。
func (s *Service) Export(ctx context.Context, q Query) ([]byte, error) {
	if err := q.Normalize(); err != nil {
		return nil, err
	}
	rows, err := s.access.Read(ctx, security.Actor(ctx), "sys:user:export", func(scoped context.Context) ([]user, error) {
		return s.repo.export(scoped, q)
	})
	if err != nil {
		return nil, err
	}
	out := [][]string{{"id", "username", "nickname", "deptId", "email", "phone", "gender", "status"}}
	for _, r := range rows {
		dept, email, phone := "", "", ""
		if r.DeptID != nil {
			dept = strconv.FormatInt(int64(*r.DeptID), 10)
		}
		if r.Email != nil {
			email = *r.Email
		}
		if r.Phone != nil {
			phone = *r.Phone
		}
		out = append(out, []string{strconv.FormatInt(int64(r.ID), 10), r.Username, r.Nickname, dept, email, phone, strconv.Itoa(r.Gender), strconv.Itoa(r.Status)})
	}
	return excel.Write(out)
}
