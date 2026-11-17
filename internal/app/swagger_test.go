package app

import (
	"encoding/json/v2"
	"os"
	"testing"
)

func TestSwaggerIDsMatchStringWireFormat(t *testing.T) {
	data, err := os.ReadFile("../../docs/swagger/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Definitions map[string]struct {
			Properties map[string]struct {
				Type  string `json:"type"`
				Items *struct {
					Type string `json:"type"`
				} `json:"items"`
			} `json:"properties"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for name, definition := range spec.Definitions {
		for field, property := range definition.Properties {
			switch field {
			case "id", "parentId", "deptId":
				checked++
				if property.Type != "string" {
					t.Fatalf("%s.%s 必须声明为字符串", name, field)
				}
			case "roleIds", "menuIds", "deptIds":
				checked++
				if property.Type != "array" || property.Items == nil || property.Items.Type != "string" {
					t.Fatalf("%s.%s 必须声明为字符串数组", name, field)
				}
			case "total":
				if property.Type != "integer" {
					t.Fatal("分页总量应保留整数")
				}
			}
		}
	}
	if checked < 15 {
		t.Fatal("缺少应生成的 ID schema")
	}
}
