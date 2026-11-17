package main

import (
	"archive/zip"
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

func (c client) generation(access string) error {
	out, err := c.request("POST", "/api/gen/tables/import", map[string]any{"tables": []string{"demo_smoke_item"}}, access, 200, 0)
	if err != nil {
		return err
	}
	var tables []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out.Data, &tables); err != nil || len(tables) != 1 || tables[0].ID == "" {
		return errors.New("生产生成器导入失败")
	}
	id := tables[0].ID
	input := map[string]any{"moduleName": "business", "featureName": "smokegen", "businessName": "smokegen", "genType": "zip", "scope": map[string]string{"mode": "admin"}, "columns": []any{map[string]any{"columnName": "name", "goField": "Name", "jsonField": "name", "isRequired": true, "isInsert": true, "isEdit": true, "isList": true, "isQuery": true, "queryType": "LIKE", "htmlType": "input"}}}
	if _, err := c.request("PUT", "/api/gen/tables/"+id, input, access, 200, 0); err != nil {
		return err
	}
	if _, err := c.request("GET", "/api/gen/tables/"+id+"/preview", nil, access, 200, 0); err != nil {
		return err
	}
	request, err := http.NewRequest("POST", c.base+"/api/gen/tables/"+id+"/generate", nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+access)
	res, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 10*1024*1024+1))
	if err != nil || res.StatusCode != 200 || res.Header.Get("Content-Type") != "application/zip" {
		return errors.New("生产生成器ZIP响应失败")
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil || len(archive.File) != 10 {
		return errors.New("生产生成器ZIP内容无效")
	}
	for _, file := range archive.File {
		if !fs.ValidPath(file.Name) || strings.HasPrefix(file.Name, "migrations/") {
			return errors.New("生产生成器ZIP路径不安全")
		}
	}
	_, err = c.request("DELETE", "/api/gen/tables/"+id, nil, access, 200, 0)
	return err
}
func (c client) runningTask(access string) error {
	out, err := c.request("POST", "/api/job/tasks", map[string]any{"name": "shutdown-smoke", "cron": "0 0 0 1 1 *", "target": "demoTask", "params": `{"message":"smoke-secret-not-for-logs","delayMs":60000}`}, access, 200, 0)
	if err != nil {
		return err
	}
	var task struct {
		ID     string `json:"id"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(out.Data, &task); err != nil || task.ID == "" || task.Status != 2 {
		return errors.New("生产任务未默认暂停")
	}
	out, err = c.request("POST", "/api/job/tasks/"+task.ID+"/run", nil, access, 200, 0)
	if err != nil {
		return err
	}
	var run struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(out.Data, &run); err != nil || run.State != "queued" {
		return errors.New("生产任务未投递")
	}
	return nil
}
