//go:build integration

package table

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/json/v2"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/project"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/security"
	"github.com/StarsSkyCat/Stars-Admin/internal/testutil"
	"gorm.io/gorm"
)

const demoDDL = `CREATE TABLE demo_generated_item(id bigint PRIMARY KEY,created_by bigint NOT NULL DEFAULT 0,updated_by bigint NOT NULL DEFAULT 0,created_at timestamp NOT NULL,updated_at timestamp NOT NULL,deleted_at timestamp,remark varchar(500),name varchar(64) NOT NULL UNIQUE,dept_id bigint,amount numeric(18,4) NOT NULL,enabled boolean NOT NULL DEFAULT false,quantity bigint NOT NULL DEFAULT 0,event_at timestamp,rank smallint NOT NULL DEFAULT 0);CREATE INDEX idx_demo_generated_item_deleted_at ON demo_generated_item(deleted_at)`

func generationService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	cfg, err := config.Load("../../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	if err := deps.DB.Exec("DROP TABLE IF EXISTS demo_generated_item;TRUNCATE gen_table,gen_table_column CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := deps.DB.Exec(demoDDL).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := deps.DB.Exec("DROP TABLE IF EXISTS demo_generated_item").Error; err != nil {
			t.Error(err)
		}
	})
	path, err := project.ModulePath()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Gen.OutputRoot = filepath.Join(t.TempDir(), "output")
	return New(deps.DB, deps.IDs, cfg.Gen, cfg.App.Env, path), deps.DB
}
func configureDemo(t *testing.T, s *Service, id model.ID, kind string, scopes ...ScopeConfig) View {
	t.Helper()
	current, err := s.Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []ColumnInput{}
	for _, c := range current.Columns {
		if common(c.ColumnName) || c.Removed {
			continue
		}
		in := c.ColumnInput
		in.IsQuery = c.ColumnName == "name" || c.ColumnName == "amount"
		in.QueryType = "EQ"
		if c.ColumnName == "name" {
			in.QueryType = "LIKE"
		}
		if c.ColumnName == "amount" {
			in.QueryType = "BETWEEN"
		}
		inputs = append(inputs, in)
	}
	scope := ScopeConfig{Mode: "mapped", DepartmentColumn: "dept_id", OwnerColumn: "created_by"}
	feature, business := "generateditem", "items"
	if len(scopes) > 0 {
		scope = scopes[0]
		feature = "generatedadmin"
		business = "adminitems"
	}
	out, err := s.Save(t.Context(), security.Principal{ID: 1, RoleIDs: []model.ID{1}}, id, Input{ModuleName: "business", FeatureName: feature, BusinessName: business, GenType: kind, Scope: scope, Columns: inputs})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func TestGeneratorImportSyncOutputAndUnsupportedSchema(t *testing.T) {
	s, db := generationService(t)
	p := security.Principal{ID: 1, RoleIDs: []model.ID{1}}
	rows, err := s.DBTables(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		if strings.HasPrefix(r.Name, "sys_") {
			t.Fatal("系统表泄漏至导入列表")
		}
		if r.Name == "demo_generated_item" {
			found = true
		}
	}
	if !found {
		t.Fatal("缺少可导入普通业务表")
	}
	if _, err := s.Import(t.Context(), p, []string{"sys_user"}); !errors.Is(err, response.ErrInvalidParam) {
		t.Fatal("允许导入系统表")
	}
	views, err := s.Import(t.Context(), p, []string{"demo_generated_item"})
	if err != nil {
		t.Fatal(err)
	}
	id := views[0].ID
	if _, err := s.Preview(t.Context(), id); !errors.Is(err, response.ErrInvalidParam) {
		t.Fatal("未显式配置范围也能生成")
	}
	configureDemo(t, s, id, "zip")
	files, err := s.Preview(t.Context(), id)
	if err != nil || len(files) != 10 {
		t.Fatalf("预览失败 %v", err)
	}
	output, err := s.Generate(t.Context(), p, id)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(output.ZIP), int64(len(output.ZIP)))
	if err != nil || len(archive.File) != 10 {
		t.Fatal("zip生成无效")
	}
	for _, f := range archive.File {
		if !fs.ValidPath(f.Name) || strings.HasPrefix(f.Name, "migrations/") {
			t.Fatal("输出执行迁移或不安全路径")
		}
	}
	s.cfg.LocalEnabled = true
	configureDemo(t, s, id, "local")
	if _, err := s.Generate(t.Context(), p, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Generate(t.Context(), p, id); !errors.Is(err, response.ErrConflict) {
		t.Fatal("local覆盖了已有文件")
	}
	s.env = "prod"
	if _, err := s.Generate(t.Context(), p, id); !errors.Is(err, response.ErrPermission) {
		t.Fatal("生产环境允许local")
	}
	s.env = "test"
	// 增删与类型变化按列名合并，类型变化不能沿用旧配置直接生成。
	if err := db.Exec("ALTER TABLE demo_generated_item ADD COLUMN description text;ALTER TABLE demo_generated_item DROP COLUMN quantity;ALTER TABLE demo_generated_item ALTER COLUMN name TYPE varchar(96)").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Preview(t.Context(), id); !errors.Is(err, response.ErrConflict) {
		t.Fatal("结构漂移未拒绝")
	}
	synced, err := s.Sync(t.Context(), p, id)
	if err != nil {
		t.Fatal(err)
	}
	changes := map[string]string{}
	for _, c := range synced.Columns {
		changes[c.ColumnName] = c.ChangeKind
	}
	if changes["description"] != "added" || changes["quantity"] != "removed" || changes["name"] != "changed" {
		t.Fatalf("同步变化标记无效 %v", changes)
	}
	if _, err := s.Preview(t.Context(), id); !errors.Is(err, response.ErrInvalidParam) {
		t.Fatal("同步后未要求重新确认配置")
	}
	configureDemo(t, s, id, "zip")
	if _, err := s.Preview(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import(t.Context(), p, []string{"demo_generated_item"}); !errors.Is(err, response.ErrConflict) {
		t.Fatal("重复导入未受唯一约束保护")
	}
	if err := s.Remove(t.Context(), p, []model.ID{id}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Table("demo_generated_item").Count(&count).Error; err != nil {
		t.Fatal("删除配置破坏了原业务表")
	}
}
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relative)
		if d.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
}
func command(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), args[0], args[1:]...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("生成物验证失败 %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return output
}
func TestGeneratedDemoCompilesLintsTestsSwaggerAndScopedCRUD(t *testing.T) {
	s, _ := generationService(t)
	p := security.Principal{ID: 1, RoleIDs: []model.ID{1}}
	views, err := s.Import(t.Context(), p, []string{"demo_generated_item"})
	if err != nil {
		t.Fatal(err)
	}
	configureDemo(t, s, views[0].ID, "zip")
	files, err := s.Preview(t.Context(), views[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	for _, dir := range []string{"internal/core", "pkg", "configs", "migrations"} {
		copyTree(t, filepath.Join(root, dir), filepath.Join(target, dir))
	}
	for _, file := range []string{"go.mod", "go.sum", ".golangci.yaml", ".swaggo"} {
		data, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, file), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range files {
		path := filepath.Join(target, file.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(file.Content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	configureDemo(t, s, views[0].ID, "zip", ScopeConfig{Mode: "admin"})
	adminFiles, err := s.Preview(t.Context(), views[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range adminFiles {
		path := filepath.Join(target, file.Path)
		if strings.HasPrefix(file.Path, "migration-drafts/") {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(file.Content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	demoDir := filepath.Join(target, "internal/modules/business/generateditem")
	if err := os.WriteFile(filepath.Join(demoDir, "acceptance_test.go"), []byte(strings.ReplaceAll(demoAcceptance, "__MODULE__", s.modulePath)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "cmd/demo"), 0700); err != nil {
		t.Fatal(err)
	}
	main := "// @title 生成器验收\n// @version 1.0\n// @BasePath /api\n// @securityDefinitions.apikey BearerAuth\n// @in header\n// @name Authorization\npackage main\nfunc main(){}\n"
	if err := os.WriteFile(filepath.Join(target, "cmd/demo/main.go"), []byte(main), 0600); err != nil {
		t.Fatal(err)
	}
	command(t, target, "go", "build", "./...")
	command(t, target, "gofmt", "-w", demoDir)
	command(t, target, "goimports", "-w", filepath.Join(demoDir, "acceptance_test.go"))
	command(t, target, "golangci-lint", "run", "./internal/modules/business/...")
	output := command(t, target, "go", "test", "-count=1", "-race", "./internal/modules/business/...")
	t.Log(string(output))
	command(t, target, "swag", "init", "-g", "main.go", "-d", "cmd/demo,internal/modules/business,internal/core", "--parseInternal", "--parseDependency", "--parseDepth", "1", "--overridesFile", ".swaggo", "-o", "swagger")
	raw, err := os.ReadFile(filepath.Join(target, "swagger/swagger.json"))
	if err != nil {
		t.Fatal(err)
	}
	var swagger struct {
		Paths map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(raw, &swagger); err != nil || len(swagger.Paths) != 6 {
		t.Fatalf("生成物Swagger无效 %v %d", err, len(swagger.Paths))
	}
	if err := os.WriteFile(filepath.Join(root, ".cache/m4-generated-demo.log"), append([]byte("go build: PASS\nlint: PASS\nswag: PASS\n"), output...), 0600); err != nil {
		t.Fatal(err)
	}
}

//go:embed testdata/demo_acceptance.txt
var demoAcceptance string
