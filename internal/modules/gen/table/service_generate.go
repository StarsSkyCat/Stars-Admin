package table

import (
	"bytes"
	"context"
	"embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/httpx"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/StarsSkyCat/Stars-Admin/pkg/safeoutput"
	"golang.org/x/mod/module"
)

//go:embed templates/*.gotmpl
var templates embed.FS

type renderField struct {
	column
	ModelType string
	ModelTag  string
	JSONTag   string
	FormTag   string
	ToTag     string
	Input     bool
	Between   bool
	ApplyCode string
	QueryCode string
	Sample    string
}
type renderData struct {
	ModulePath       string
	Module           string
	Feature          string
	Business         string
	Table            string
	PermissionPrefix string
	Fields           []renderField
	AdminOnly        bool
	DepartmentColumn string
	OwnerColumn      string
	MatchCode        string
	Helpers          string
}

// Preview 完整校验当前结构与配置后渲染固定模板。
func (s *Service) Preview(ctx context.Context, id model.ID) ([]GeneratedFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.preview(ctx, id)
}
func (s *Service) preview(ctx context.Context, id model.ID) ([]GeneratedFile, error) {
	row, err := s.repo.get(ctx, nil, id)
	if err != nil {
		return nil, httpx.DBError(err)
	}
	if !s.allowed(row.TableNameValue) {
		return nil, response.ErrPermission
	}
	cols, err := s.repo.columns(ctx, nil, id, true)
	if err != nil {
		return nil, err
	}
	var opts options
	if err := json.Unmarshal([]byte(row.Options), &opts); err != nil {
		return nil, err
	}
	if err := validateSchema(row, cols, opts); err != nil {
		return nil, err
	}
	fresh, err := s.repo.describe(ctx, nil, s.cfg.Schema, row.TableNameValue)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(fresh, opts.Snapshot) {
		return nil, response.ErrConflict
	}
	return render(s.modulePath, row, cols, opts)
}
func render(modulePath string, row table, cols []column, opts options) ([]GeneratedFile, error) {
	if err := module.CheckPath(modulePath); err != nil {
		return nil, response.ErrInvalidParam
	}
	data := renderData{ModulePath: modulePath, Module: row.ModuleName, Feature: row.FeatureName, Business: row.BusinessName, Table: row.TableNameValue, PermissionPrefix: row.ModuleName + ":" + row.BusinessName + ":", AdminOnly: opts.Scope.Mode == "admin", DepartmentColumn: opts.Scope.DepartmentColumn, OwnerColumn: opts.Scope.OwnerColumn}
	metadata := map[string]MetadataColumn{}
	for _, m := range opts.Snapshot.Columns {
		metadata[m.Name] = m
	}
	query, numeric := false, false
	for _, c := range cols {
		if c.DeletedAt.Valid || common(c.ColumnName) {
			continue
		}
		f := renderField{column: c, ModelType: c.GoType, ModelTag: "gorm:\"column:" + c.ColumnName + "\"", JSONTag: "json:\"" + c.JSONField + "\"", FormTag: "form:\"" + c.JSONField + "\"", ToTag: "form:\"" + c.JSONField + "To\"", Input: c.IsInsert || c.IsEdit, Between: c.QueryType == "BETWEEN"}
		if c.IsNullable {
			f.ModelType = "*" + c.GoType
		}
		switch c.GoType {
		case "string":
			f.Sample = strconv.Quote("x")
			if metadata[c.ColumnName].UDT == "numeric" {
				f.Sample = strconv.Quote("1")
				numeric = true
			}
		case "bool":
			f.Sample = "true"
		case "time.Time":
			f.Sample = "time.Now().UTC()"
		case "model.ID":
			f.Sample = "model.ID(1)"
		case "int64":
			f.Sample = "int64(1)"
		default:
			f.Sample = "int(1)"
		}
		f.ApplyCode = applyCode(c, metadata[c.ColumnName])
		if c.IsQuery {
			query = true
			f.QueryCode = queryCode(c, metadata[c.ColumnName])
		}
		data.Fields = append(data.Fields, f)
	}
	data.MatchCode = matchCode(opts.Scope, cols)
	if numeric || query {
		data.Helpers = decimalHelper
	}
	if query {
		data.Helpers += filterHelper
	}
	base := "internal/modules/" + row.ModuleName + "/" + row.FeatureName + "/"
	files := []GeneratedFile{}
	for _, name := range []string{"model", "dto", "repository", "service", "handler", "routes", "service_test"} {
		raw, err := templates.ReadFile("templates/" + name + ".gotmpl")
		if err != nil {
			return nil, err
		}
		tpl, err := template.New(name).Funcs(template.FuncMap{"q": strconv.Quote, "tag": func(s string) string { return "`" + s + "`" }}).Parse(string(raw))
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if err := tpl.Execute(&out, data); err != nil {
			return nil, err
		}
		path := base + name + ".go"
		formatted, err := formatGenerated(path, out.Bytes())
		if err != nil {
			return nil, fmt.Errorf("生成的 Go 结构无效: %w", err)
		}
		files = append(files, GeneratedFile{path, string(formatted)})
	}
	draft := "migration-drafts/create_" + row.TableNameValue
	files = append(files, GeneratedFile{draft + ".up.sql", draftSQL(opts.Snapshot)}, GeneratedFile{draft + ".down.sql", "-- 仅供隔离库 review，禁止对已有业务表直接执行。\nDROP TABLE IF EXISTS " + quoteSQL(row.TableNameValue) + ";\n"}, GeneratedFile{base + "ASSEMBLY.md", assembly(data)})
	return files, nil
}
func applyCode(c column, m MetadataColumn) string {
	validate := ""
	v := "*in." + c.GoField
	switch c.GoType {
	case "model.ID":
		validate = "if " + v + "<=0{return nil,response.ErrInvalidParam};"
	case "int":
		bits := 32
		if m.UDT == "int2" {
			bits = 16
		}
		lower, upper := int64(-2147483648), int64(2147483647)
		if bits == 16 {
			lower, upper = -32768, 32767
		}
		validate = fmt.Sprintf("if %s < %d || %s > %d{return nil,response.ErrInvalidParam};", v, lower, v, upper)
	case "string":
		maxLen := 4096
		if c.MaxLength != nil {
			maxLen = *c.MaxLength
		}
		validate = fmt.Sprintf("if utf8.RuneCountInString(%s)>%d{return nil,response.ErrInvalidParam};", v, maxLen)
		if m.UDT == "numeric" {
			p, scale := 0, -1
			if m.Precision != nil {
				p = *m.Precision
			}
			if m.Scale != nil {
				scale = *m.Scale
			}
			validate += fmt.Sprintf("if !validDecimal(%s,%d,%d){return nil,response.ErrInvalidParam};", v, p, scale)
		}
	}
	assign := "row." + c.GoField + "=*in." + c.GoField + ";"
	if c.IsNullable {
		assign = "row." + c.GoField + "=in." + c.GoField + ";"
	}
	if c.GoType == "time.Time" {
		if c.IsNullable {
			assign = "row." + c.GoField + "=new(in." + c.GoField + ".UTC());"
		} else {
			assign = "row." + c.GoField + "=in." + c.GoField + ".UTC();"
		}
	}
	body := "if in." + c.GoField + "!=nil{" + validate + assign + "if create{columns=append(columns," + strconv.Quote(c.ColumnName) + ")}}"
	missing := ""
	if !m.Nullable && (m.Default == nil || c.IsRequired) {
		missing += "if create{return nil,response.ErrInvalidParam};"
	}
	if c.IsNullable {
		missing += "row." + c.GoField + "=nil;"
		if m.Default == nil {
			missing += "if create{columns=append(columns," + strconv.Quote(c.ColumnName) + ")};"
		}
	}
	if missing != "" {
		body += "else{" + missing + "}"
	}
	create, edit := "", ""
	if c.IsInsert {
		create = body
	} else {
		create = "if in." + c.GoField + "!=nil{return nil,response.ErrInvalidParam}"
	}
	if c.IsEdit {
		edit = body
	} else {
		edit = "if in." + c.GoField + "!=nil{return nil,response.ErrInvalidParam}"
	}
	return "if create{" + create + "}else{" + edit + "}"
}
func queryCode(c column, m MetadataColumn) string {
	maxLen, p, scale := 4096, 0, -1
	if c.MaxLength != nil {
		maxLen = *c.MaxLength
	}
	if m.Precision != nil {
		p = *m.Precision
	}
	if m.Scale != nil {
		scale = *m.Scale
	}
	kind := m.UDT
	if c.GoType == "model.ID" {
		kind = "id"
	}
	parse := func(value string) string {
		return fmt.Sprintf("parseFilter(%s,%q,%d,%d,%d)", value, kind, maxLen, p, scale)
	}
	f := "q." + c.GoField
	code := "{if " + f + "!=\"\""
	if c.QueryType == "BETWEEN" {
		code += "||" + f + "To!=\"\""
	}
	code += "{"
	if c.QueryType == "BETWEEN" {
		code += "if " + f + "==\"\"||" + f + "To==\"\"{return response.ErrInvalidParam};"
	}
	code += "v,err:=" + parse(f) + ";if err!=nil{return err};"
	if c.QueryType == "LIKE" {
		code += `text,ok:=v.(string);if !ok{return response.ErrInvalidParam};v="%"+text+"%";`
	}
	values := "[]any{v}"
	if c.QueryType == "BETWEEN" {
		code += "to,err:=" + parse(f+"To") + ";if err!=nil{return err};"
		values = "[]any{v,to}"
	}
	return code + "filters=append(filters,filter{" + strconv.Quote(c.ColumnName) + "," + strconv.Quote(c.QueryType) + "," + values + "})}}"
}
func matchCode(scope ScopeConfig, cols []column) string {
	if scope.Mode == "admin" {
		return ""
	}
	lookup := func(name string) (string, bool) {
		if name == "created_by" {
			return "row.CreatedBy", false
		}
		if name == "updated_by" {
			return "row.UpdatedBy", false
		}
		if name == "id" {
			return "row.ID", false
		}
		for _, c := range cols {
			if c.ColumnName == name {
				return "row." + c.GoField, c.IsNullable
			}
		}
		return "", false
	}
	code := ""
	if scope.DepartmentColumn != "" {
		field, nullable := lookup(scope.DepartmentColumn)
		if nullable {
			code += "if " + field + "!=nil&&scope.AllowsDepartment(*" + field + "){return true};"
		} else {
			code += "if scope.AllowsDepartment(" + field + "){return true};"
		}
	}
	if scope.OwnerColumn != "" {
		field, nullable := lookup(scope.OwnerColumn)
		if nullable {
			code += "if scope.Self&&" + field + "!=nil&&*" + field + "==scope.UserID{return true}"
		} else {
			code += "if scope.Self&&" + field + "==scope.UserID{return true}"
		}
	}
	return code
}
func quoteSQL(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }
func draftSQL(s Snapshot) string {
	parts := []string{}
	for _, c := range s.Columns {
		definition := quoteSQL(c.Name) + " " + c.DBType
		if !c.Nullable {
			definition += " NOT NULL"
		}
		if c.Default != nil {
			definition += " DEFAULT " + *c.Default
		}
		parts = append(parts, "  "+definition)
	}
	for _, c := range s.Constraints {
		parts = append(parts, "  CONSTRAINT "+c)
	}
	out := "-- 迁移参考稿：原表已存在，禁止加入正式 migrations 或重复执行。\nCREATE TABLE " + quoteSQL(s.Name) + " (\n" + strings.Join(parts, ",\n") + "\n);\n"
	for _, index := range s.Indexes {
		out += index + ";\n"
	}
	return out
}
func assembly(d renderData) string {
	return fmt.Sprintf("# %s 装配说明\n\n这是待审代码，原表已存在，不执行 migration-drafts。\n\n1. review 业务校验、唯一约束、关联解绑及字段展示；复制本目录至工程相同路径。\n2. 在已有模块 module.go 手工构造 New(db, ids, access, scopes)，然后 NewHandler(service).RegisterRoutes(rg.Group(%q), guards)。缺失授权与范围能力默认拒绝。access 必须提供与系统授权变更共享的 Read/Allow 门禁；scopes 使用可信业务 Resolver，不能接收 HTTP 范围。\n3. routes 的 guards 必须注入 Required、Permit 与操作审计；首登改密和有效会话校验不能省略。\n4. 菜单权限逐项配置：%slist/get/add/edit/remove；默认范围模式 %s。配置菜单不得自动扩大普通角色范围。\n5. gofmt/goimports、make lint/test/swag 与隔离 PG CRUD/越权测试通过后才能接入。\n\n模型保留 NULL；numeric 使用十进制字符串，不接受指数格式。新建缺少有默认值的非空字段时由数据库填充；PUT 中省略非空字段保留原值，省略可空字段视为 NULL。ID 均使用字符串；业务唯一键软删除后不复用。\n", d.Feature, "/"+d.Business, d.PermissionPrefix, map[bool]string{true: "admin", false: "mapped"}[d.AdminOnly])
}

// Generate 默认返回 zip；local 由保存配置及当前环境双重限制。
func (s *Service) Generate(ctx context.Context, p security.Principal, id model.ID) (Generated, error) {
	if !p.Admin() {
		return Generated{}, response.ErrPermission
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.repo.get(ctx, nil, id)
	if err != nil {
		return Generated{}, httpx.DBError(err)
	}
	if row.GenType == "local" && (!s.cfg.LocalEnabled || s.env == "prod") {
		return Generated{}, response.ErrPermission
	}
	files, err := s.preview(ctx, id)
	if err != nil {
		return Generated{}, err
	}
	output := make([]safeoutput.File, 0, len(files))
	paths := []string{}
	for _, f := range files {
		output = append(output, safeoutput.File{Path: f.Path, Content: f.Content})
		paths = append(paths, f.Path)
	}
	if row.GenType == "local" {
		if err := safeoutput.Local(s.cfg.OutputRoot, output); err != nil {
			if errors.Is(err, os.ErrExist) || errors.Is(err, os.ErrPermission) {
				return Generated{}, response.ErrConflict
			}
			return Generated{}, err
		}
		return Generated{Local: &GenerateResult{paths}}, nil
	}
	data, err := safeoutput.ZIP(output)
	return Generated{ZIP: data}, err
}
