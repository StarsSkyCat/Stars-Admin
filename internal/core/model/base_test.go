package model

import (
	"encoding/json/v2"
	"strings"
	"testing"
	"time"
)

func TestIDStringsAndUTC(t *testing.T) {
	value := struct {
		ID  ID        `json:"id"`
		IDs []ID      `json:"ids"`
		At  time.Time `json:"at"`
	}{ID: 9223372036854775807, IDs: []ID{9007199254740993}, At: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"id":"9223372036854775807"`, `"ids":["9007199254740993"]`, `"at":"2026-11-17T00:00:00Z"`} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("序列化缺少 %s", expected)
		}
	}
	var id ID
	if err := json.Unmarshal([]byte(`"9223372036854775807"`), &id); err != nil || id != 9223372036854775807 {
		t.Fatal("ID 精度丢失")
	}
	for _, bad := range []string{`9007199254740993`, `"9223372036854775808"`, `"-1"`, `"abc"`, `null`} {
		if json.Unmarshal([]byte(bad), &id) == nil {
			t.Fatalf("非法 ID 未拒绝：%s", bad)
		}
	}
}
