package table

import (
	"context"
	"encoding/json/v2"
	"go/token"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"gorm.io/gorm"
)

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
var fieldName = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]{0,63}$`)
var jsonName = regexp.MustCompile(`^[a-z][a-zA-Z0-9]{0,63}$`)

func validIdentifier(s string) bool {
	return identifier.MatchString(s) && !token.Lookup(s).IsKeyword() && !slices.Contains([]string{"internal", "core", "pkg", "cmd", "config", "configs", "migrations", "vendor", "templates", "app", "runtime", "unsafe", "test", "init", "main", "nil", "true", "false", "iota"}, s)
}
func names(name string) (string, string) {
	parts := strings.Split(name, "_")
	var goField, js string
	for i, p := range parts {
		if p == "" {
			continue
		}
		title := strings.ToUpper(p[:1]) + p[1:]
		if p == "id" {
			goField += "ID"
		} else {
			goField += title
		}
		if i == 0 {
			js = p
		} else {
			js += title
		}
	}
	return goField, js
}
func goType(c MetadataColumn) string {
	switch c.UDT {
	case "int8":
		if c.Name == "id" || strings.HasSuffix(c.Name, "_id") || c.Name == "created_by" || c.Name == "updated_by" {
			return "model.ID"
		}
		return "int64"
	case "int4", "int2":
		return "int"
	case "varchar", "text", "bpchar", "numeric":
		return "string"
	case "timestamp":
		return "time.Time"
	case "bool":
		return "bool"
	default:
		return ""
	}
}
func (s *Service) merge(ctx context.Context, tx *gorm.DB, p security.Principal, id model.ID, old []column, metadata []MetadataColumn) ([]column, error) {
	byName := map[string]column{}
	for _, c := range old {
		byName[c.ColumnName] = c
	}
	out := []column{}
	present := map[string]bool{}
	for _, m := range metadata {
		present[m.Name] = true
		c, exists := byName[m.Name]
		if !exists {
			n, err := s.ids.Next()
			if err != nil {
				return nil, err
			}
			gf, jf := names(m.Name)
			business := !common(m.Name)
			c = column{ID: model.ID(n), CreatedBy: p.ID, TableID: id, ColumnName: m.Name, GoField: gf, JSONField: jf, IsInsert: business, IsEdit: business, IsList: business, IsRequired: business && !m.Nullable && m.Default == nil, QueryType: "EQ", HTMLType: "input", ChangeKind: "added"}
		} else {
			c.ChangeKind = "same"
			if c.DBType != m.DBType || c.IsNullable != m.Nullable || c.IsPK != m.Primary {
				c.ChangeKind = "changed"
			}
			if c.DeletedAt.Valid {
				c.ChangeKind = "added"
			}
			c.DeletedAt = gorm.DeletedAt{}
		}
		c.DBType = m.DBType
		c.GoType = goType(m)
		c.IsNullable = m.Nullable
		c.IsPK = m.Primary
		c.ColumnComment = truncate(m.Comment, 255)
		c.MaxLength = m.MaxLength
		c.NumericPrecision = m.Precision
		c.NumericScale = m.Scale
		c.ColumnDefault = m.Default
		c.Sort = m.Position
		c.UpdatedBy = p.ID
		if err := s.repo.saveColumn(ctx, tx, &c, !exists); err != nil {
			return nil, httpx.DBError(err)
		}
		out = append(out, c)
	}
	for _, c := range old {
		if !present[c.ColumnName] {
			c.ChangeKind = "removed"
			c.DeletedAt = gorm.DeletedAt{Time: nowUTC(), Valid: true}
			c.UpdatedBy = p.ID
			if err := s.repo.saveColumn(ctx, tx, &c, false); err != nil {
				return nil, err
			}
			out = append(out, c)
		}
	}
	return out, nil
}
func nowUTC() time.Time { return time.Now().UTC() }

// Save 仅接受已导入列配置，受保护字段的生成行为不可开启。
func (s *Service) Save(ctx context.Context, p security.Principal, id model.ID, in Input) (View, error) {
	if !p.Admin() {
		return View{}, response.ErrPermission
	}
	if !validIdentifier(in.ModuleName) || !validIdentifier(in.FeatureName) || !validIdentifier(in.BusinessName) || (in.GenType != "zip" && in.GenType != "local") || len([]rune(in.Author)) > 64 {
		return View{}, response.ErrInvalidParam
	}
	if in.GenType == "local" && (!s.cfg.LocalEnabled || s.env == "prod") {
		return View{}, response.ErrPermission
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out View
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := s.repo.get(ctx, tx, id)
		if err != nil {
			return httpx.DBError(err)
		}
		cols, err := s.repo.columns(ctx, tx, id, true)
		if err != nil {
			return err
		}
		var opts options
		if err := json.Unmarshal([]byte(row.Options), &opts); err != nil {
			return err
		}
		mapping := map[string]ColumnInput{}
		for _, c := range in.Columns {
			if _, ok := mapping[c.ColumnName]; ok {
				return response.ErrInvalidParam
			}
			mapping[c.ColumnName] = c
		}
		for i := range cols {
			c := &cols[i]
			v, ok := mapping[c.ColumnName]
			if !ok {
				continue
			}
			delete(mapping, c.ColumnName)
			if c.DeletedAt.Valid || common(c.ColumnName) {
				return response.ErrInvalidParam
			}
			if !fieldName.MatchString(v.GoField) || !jsonName.MatchString(v.JSONField) {
				return response.ErrInvalidParam
			}
			if !slices.Contains([]string{"EQ", "NE", "LIKE", "BETWEEN"}, v.QueryType) {
				return response.ErrInvalidParam
			}
			if v.HTMLType == "" {
				v.HTMLType = "input"
			}
			if !slices.Contains([]string{"input", "select", "radio", "datetime"}, v.HTMLType) {
				return response.ErrInvalidParam
			}
			c.GoField = v.GoField
			c.JSONField = v.JSONField
			c.IsRequired = v.IsRequired
			c.IsInsert = v.IsInsert
			c.IsEdit = v.IsEdit
			c.IsList = v.IsList
			c.IsQuery = v.IsQuery
			c.QueryType = v.QueryType
			c.HTMLType = v.HTMLType
			c.DictType = v.DictType
			c.UpdatedBy = p.ID
		}
		if len(mapping) > 0 {
			return response.ErrInvalidParam
		}
		row.ModuleName = in.ModuleName
		row.FeatureName = in.FeatureName
		row.BusinessName = in.BusinessName
		row.Author = in.Author
		row.GenType = in.GenType
		row.UpdatedBy = p.ID
		opts.Scope = in.Scope
		opts.Configured = true
		if err := validateSchema(row, cols, opts); err != nil {
			return err
		}
		for _, c := range cols {
			if err := s.repo.saveColumn(ctx, tx, new(c), false); err != nil {
				return err
			}
		}
		raw, err := json.Marshal(opts)
		if err != nil {
			return err
		}
		row.Options = string(raw)
		if err := s.repo.save(ctx, tx, &row, false); err != nil {
			return err
		}
		out = view(row, cols)
		return nil
	})
	return out, err
}
func validateSchema(row table, cols []column, opts options) error {
	if !opts.Configured || !validIdentifier(row.ModuleName) || !validIdentifier(row.FeatureName) || !validIdentifier(row.BusinessName) {
		return response.ErrInvalidParam
	}
	if opts.Scope.Mode != "admin" && opts.Scope.Mode != "mapped" {
		return response.ErrInvalidParam
	}
	metadata := map[string]MetadataColumn{}
	pk := 0
	for _, c := range opts.Snapshot.Columns {
		metadata[c.Name] = c
		if c.Primary {
			pk++
		}
		if !dbName.MatchString(c.Name) || c.Identity || c.Generated || goType(c) == "" {
			return response.ErrInvalidParam
		}
		// 名称拦截是保守防线，业务语义仍须开发者在接入前人工审核。
		for _, word := range []string{"password", "passwd", "pwd", "secret", "token", "captcha", "credential", "hash", "private_key", "api_key", "apikey"} {
			if strings.Contains(c.Name, word) {
				return response.ErrInvalidParam
			}
		}
	}
	id, ok := metadata["id"]
	if pk != 1 || !ok || !id.Primary || id.UDT != "int8" || id.Nullable || id.Default != nil {
		return response.ErrInvalidParam
	}
	for name, udt := range map[string]string{"created_by": "int8", "updated_by": "int8", "created_at": "timestamp", "updated_at": "timestamp", "deleted_at": "timestamp", "remark": "varchar"} {
		c, ok := metadata[name]
		if !ok || c.UDT != udt {
			return response.ErrInvalidParam
		}
		if c.Nullable != (name == "deleted_at" || name == "remark") {
			return response.ErrInvalidParam
		}
	}
	if opts.Scope.Mode == "mapped" {
		if opts.Scope.DepartmentColumn == "" && opts.Scope.OwnerColumn == "" {
			return response.ErrInvalidParam
		}
		for _, name := range []string{opts.Scope.DepartmentColumn, opts.Scope.OwnerColumn} {
			if name != "" {
				c, ok := metadata[name]
				if !ok || c.UDT != "int8" || goType(c) != "model.ID" {
					return response.ErrInvalidParam
				}
			}
		}
	} else if opts.Scope.DepartmentColumn != "" || opts.Scope.OwnerColumn != "" {
		return response.ErrInvalidParam
	}
	goNames := map[string]bool{}
	jsonNames := map[string]bool{}
	for _, name := range []string{"ID", "CreatedBy", "UpdatedBy", "CreatedAt", "UpdatedAt", "DeletedAt", "Remark", "TableName", "Base", "Meta", "Input"} {
		goNames[name] = true
	}
	for _, name := range []string{"id", "createdBy", "updatedBy", "createdAt", "updatedAt", "deletedAt", "remark"} {
		jsonNames[name] = true
	}
	active := 0
	for _, c := range cols {
		if c.DeletedAt.Valid || common(c.ColumnName) {
			continue
		}
		active++
		m, ok := metadata[c.ColumnName]
		if !ok || c.GoType != goType(m) || !fieldName.MatchString(c.GoField) || !jsonName.MatchString(c.JSONField) || goNames[c.GoField] || jsonNames[c.JSONField] {
			return response.ErrInvalidParam
		}
		goNames[c.GoField] = true
		jsonNames[c.JSONField] = true
		if !m.Nullable && m.Default == nil && !c.IsInsert {
			return response.ErrInvalidParam
		}
		if c.IsRequired && (!c.IsInsert || m.Nullable) {
			return response.ErrInvalidParam
		}
		if c.IsQuery && c.QueryType == "LIKE" && !slices.Contains([]string{"varchar", "text", "bpchar"}, m.UDT) {
			return response.ErrInvalidParam
		}
		if c.IsQuery && c.QueryType == "BETWEEN" && c.GoType == "bool" {
			return response.ErrInvalidParam
		}
		if m.UDT == "numeric" && m.Scale != nil && *m.Scale < 0 {
			return response.ErrInvalidParam
		}
	}
	if active == 0 {
		return response.ErrInvalidParam
	}
	return nil
}
