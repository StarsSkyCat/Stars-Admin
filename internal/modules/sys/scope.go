package sys

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

type scopeResolver struct{ *features }

func (r scopeResolver) Resolve(ctx context.Context, tx *gorm.DB, p security.Principal) (security.Scope, error) {
	out := security.Scope{UserID: p.ID}
	if p.ID <= 0 {
		return out, nil
	}
	var identityErr error
	p, identityErr = r.users.ScopeIdentity(ctx, tx, p)
	if identityErr != nil {
		return out, identityErr
	}
	if p.MustChangePassword {
		return out, response.ErrPasswordChange
	}
	if p.Admin() {
		out.All = true
		return out, nil
	}
	department, err := r.users.DepartmentID(ctx, tx, p.ID)
	if err != nil {
		return out, err
	}
	out.DepartmentID = department
	rules, err := r.roles.ScopeRules(ctx, tx, p.RoleIDs)
	if err != nil {
		return out, err
	}
	for _, rule := range rules {
		switch rule.Kind {
		case 1:
			out.All = true
		case 2:
			out.Departments = append(out.Departments, rule.Departments...)
		case 3:
			if department != nil {
				valid, e := r.depts.Descendants(ctx, tx, *department)
				if e != nil {
					return out, e
				}
				if len(valid) > 0 {
					out.Departments = append(out.Departments, *department)
				}
			}
		case 4:
			if department != nil {
				ids, e := r.depts.Descendants(ctx, tx, *department)
				if e != nil {
					return out, e
				}
				out.Departments = append(out.Departments, ids...)
			}
		case 5:
			out.Self = true
		}
	}
	return out, nil
}

var _ security.Resolver = scopeResolver{}
