package app

import (
	"encoding/json/v2"
	"strings"
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

func TestSwaggerTokenResponsesMatchAuthenticationWireContract(t *testing.T) {
	type schema struct {
		Ref        string            `json:"$ref"`
		Type       string            `json:"type"`
		Properties map[string]schema `json:"properties"`
	}
	var doc struct {
		Definitions map[string]schema `json:"definitions"`
		Paths       map[string]map[string]struct {
			Responses map[string]struct {
				Schema schema `json:"schema"`
			} `json:"responses"`
		} `json:"paths"`
	}
	raw, err := swag.ReadDoc()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	resolve := func(s schema) schema {
		if name, ok := strings.CutPrefix(s.Ref, "#/definitions/"); ok {
			return doc.Definitions[name]
		}
		return s
	}
	for _, path := range []string{"/sys/auth/login", "/sys/auth/refresh"} {
		body := resolve(doc.Paths[path]["post"].Responses["200"].Schema)
		data := resolve(body.Properties["data"])
		if data.Type != "object" || len(data.Properties) != 5 {
			t.Fatalf("%s 令牌响应被解析成错误的同名依赖类型", path)
		}
		for name, kind := range map[string]string{"accessToken": "string", "refreshToken": "string", "tokenType": "string", "expiresIn": "integer", "mustChangePassword": "boolean"} {
			if data.Properties[name].Type != kind {
				t.Fatalf("%s 缺少正确的 %s 类型", path, name)
			}
		}
	}
}
