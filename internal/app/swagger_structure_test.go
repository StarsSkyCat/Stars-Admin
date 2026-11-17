package app

import (
	"encoding/json/v2"
	"testing"

	"github.com/swaggo/swag"
)

func TestSwaggerStructureAndProtectedOperations(t *testing.T) {
	raw, err := swag.ReadDoc()
	if err != nil {
		t.Fatal(err)
	}
	type operation struct {
		Security  []map[string][]string `json:"security"`
		Responses map[string]any        `json:"responses"`
	}
	var doc struct {
		Swagger             string                          `json:"swagger"`
		BasePath            string                          `json:"basePath"`
		Paths               map[string]map[string]operation `json:"paths"`
		Definitions         map[string]any                  `json:"definitions"`
		SecurityDefinitions map[string]struct {
			Type string `json:"type"`
			Name string `json:"name"`
			In   string `json:"in"`
		} `json:"securityDefinitions"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Swagger != "2.0" || doc.BasePath != "/api" || len(doc.Definitions) == 0 {
		t.Fatal("Swagger 结构或基础路径错误")
	}
	auth := doc.SecurityDefinitions["BearerAuth"]
	if auth.Type != "apiKey" || auth.In != "header" || auth.Name != "Authorization" {
		t.Fatal("BearerAuth 声明缺失")
	}
	for _, path := range []string{"/sys/users", "/sys/roles", "/job/tasks", "/job/logs", "/gen/tables/{id}/generate"} {
		ops := doc.Paths[path]
		if len(ops) == 0 {
			t.Fatalf("缺少核心接口 %s", path)
		}
		for method, op := range ops {
			if len(op.Security) == 0 || len(op.Responses) == 0 {
				t.Fatalf("受保护接口未声明授权或响应 %s %s", method, path)
			}
		}
	}
}
