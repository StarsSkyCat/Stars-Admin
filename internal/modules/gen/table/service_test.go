package table

import (
	"errors"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
)

func fixture() (table, []column, options) {
	metadata := []MetadataColumn{{Name: "id", UDT: "int8", DBType: "bigint", Primary: true}, {Name: "created_by", UDT: "int8", DBType: "bigint"}, {Name: "updated_by", UDT: "int8", DBType: "bigint"}, {Name: "created_at", UDT: "timestamp", DBType: "timestamp without time zone"}, {Name: "updated_at", UDT: "timestamp", DBType: "timestamp without time zone"}, {Name: "deleted_at", UDT: "timestamp", DBType: "timestamp without time zone", Nullable: true}, {Name: "remark", UDT: "varchar", DBType: "character varying(500)", Nullable: true, MaxLength: new(500)}, {Name: "name", UDT: "varchar", DBType: "character varying(64)", MaxLength: new(64)}, {Name: "amount", UDT: "numeric", DBType: "numeric(18,4)", Precision: new(18), Scale: new(4)}, {Name: "dept_id", UDT: "int8", DBType: "bigint", Nullable: true}, {Name: "enabled", UDT: "bool", DBType: "boolean", Default: new("false")}}
	row := table{ID: 1, TableNameValue: "demo_item", ModuleName: "business", FeatureName: "item", BusinessName: "items", GenType: "zip"}
	cols := []column{}
	for _, m := range metadata {
		g, j := names(m.Name)
		cols = append(cols, column{ID: model.ID(len(cols) + 1), ColumnName: m.Name, GoField: g, JSONField: j, GoType: goType(m), DBType: m.DBType, IsPK: m.Primary, IsNullable: m.Nullable, MaxLength: m.MaxLength, NumericPrecision: m.Precision, NumericScale: m.Scale, ColumnDefault: m.Default, IsInsert: !common(m.Name), IsEdit: !common(m.Name), IsList: !common(m.Name), IsQuery: m.Name == "name", QueryType: "LIKE"})
	}
	opts := options{Snapshot: Snapshot{Name: "demo_item", Columns: metadata}, Configured: true, Scope: ScopeConfig{Mode: "mapped", DepartmentColumn: "dept_id", OwnerColumn: "created_by"}}
	return row, cols, opts
}
func TestGeneratedTemplatesContainRealBehaviorAndSafeTypes(t *testing.T) {
	row, cols, opts := fixture()
	if err := validateSchema(row, cols, opts); err != nil {
		t.Fatal(err)
	}
	files, err := render("github.com/StarsSkyCat/Stars-Admin", row, cols, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 10 {
		t.Fatalf("缺少生成物 %d", len(files))
	}
	for _, file := range files {
		if strings.HasSuffix(file.Path, ".go") {
			if _, err := parser.ParseFile(token.NewFileSet(), file.Path, file.Content, parser.AllErrors); err != nil {
				t.Fatal(err)
			}
		}
		if strings.HasSuffix(file.Path, "model.go") && (strings.Contains(file.Content, "float64") || !strings.Contains(file.Content, "*model.ID")) {
			t.Fatal("精度或可空性丢失")
		}
		if strings.HasSuffix(file.Path, "service_test.go") && !strings.Contains(file.Content, "TestServiceEmptyScopeDeniesData") {
			t.Fatal("缺少越权行为测试")
		}
	}
}
func TestGeneratorRefusesUnsupportedSchemaAndMissingScope(t *testing.T) {
	for _, mode := range []string{"scope", "composite", "unknown", "missing", "keyword", "shadow"} {
		t.Run(mode, func(t *testing.T) {
			row, cols, opts := fixture()
			switch mode {
			case "scope":
				opts.Scope = ScopeConfig{}
			case "composite":
				opts.Snapshot.Columns[7].Primary = true
			case "unknown":
				opts.Snapshot.Columns[7].UDT = "jsonb"
			case "missing":
				opts.Snapshot.Columns = opts.Snapshot.Columns[1:]
			case "keyword":
				row.FeatureName = "type"
			case "shadow":
				cols[7].GoField = "ID"
			}
			if err := validateSchema(row, cols, opts); !errors.Is(err, response.ErrInvalidParam) {
				t.Fatal("不支持结构被静默生成")
			}
		})
	}
}

func TestRendererWorksWithoutGoToolchainOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	row, cols, opts := fixture()
	for _, mode := range []string{"mapped", "admin"} {
		opts.Scope = ScopeConfig{Mode: mode}
		if mode == "mapped" {
			opts.Scope.DepartmentColumn = "dept_id"
			opts.Scope.OwnerColumn = "created_by"
		}
		if _, err := render("github.com/StarsSkyCat/Stars-Admin", row, cols, opts); err != nil {
			t.Fatalf("精简运行环境生成失败 %s: %v", mode, err)
		}
	}
}

func TestGeneratorRejectsCredentialColumnVariants(t *testing.T) {
	for _, name := range []string{"password", "passwd", "pwd", "pwd_hash", "password_digest", "access_token", "api_key", "private_key", "credential", "hash"} {
		t.Run(name, func(t *testing.T) {
			row, cols, opts := fixture()
			opts.Snapshot.Columns[7].Name = name
			if err := validateSchema(row, cols, opts); !errors.Is(err, response.ErrInvalidParam) {
				t.Fatal("敏感凭证列变体未拒绝")
			}
		})
	}
}
