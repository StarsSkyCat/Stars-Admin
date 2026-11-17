// Package main 在隔离生产冒烟环境验证认证流程，不打印密码或令牌。
package main

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const newPassword = "Smoke-Changed-Password-456"

type envelope struct {
	Code int            `json:"code"`
	Data jsontext.Value `json:"data"`
}
type tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	MustChange   bool   `json:"mustChangePassword"`
}
type client struct {
	base string
	http *http.Client
}

func (c client) request(method, path string, body any, access string, wantStatus, wantCode int) (envelope, error) {
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return envelope{}, err
		}
	}
	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(data))
	if err != nil {
		return envelope{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if access != "" {
		req.Header.Set("Authorization", "Bearer "+access)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return envelope{}, err
	}
	defer func() { _ = res.Body.Close() }()
	var out envelope
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, errors.New("服务没有返回统一 JSON")
	}
	if res.StatusCode != wantStatus || out.Code != wantCode {
		return out, fmt.Errorf("%s %s 返回 %d/%d，期望 %d/%d", method, path, res.StatusCode, out.Code, wantStatus, wantCode)
	}
	return out, nil
}
func (c client) login(password string) (tokens, error) {
	out, err := c.request("POST", "/api/sys/auth/login", map[string]string{"username": "admin", "password": password}, "", 200, 0)
	if err != nil {
		return tokens{}, err
	}
	var result tokens
	err = json.Unmarshal(out.Data, &result)
	return result, err
}
func run() error {
	c := client{os.Getenv("STARS_SMOKE_URL"), &http.Client{Timeout: 5 * time.Second}}
	if c.base == "" {
		return errors.New("必须指定隔离服务 STARS_SMOKE_URL")
	}
	if len(os.Args) > 1 && os.Args[1] == "after-seed" {
		pair, err := c.login(newPassword)
		if err != nil {
			return err
		}
		if pair.MustChange {
			return errors.New("重复 seed 恢复了首登标识")
		}
		return nil
	}
	if _, err := c.request("GET", "/api/sys/users", nil, "", 401, 2000); err != nil {
		return err
	}
	first, err := c.login(os.Getenv("STARS_SEED_ADMIN_PASSWORD"))
	if err != nil {
		return err
	}
	if !first.MustChange {
		return errors.New("首次登录缺少改密标识")
	}
	if _, err := c.request("GET", "/api/sys/users", nil, first.AccessToken, 403, 2008); err != nil {
		return err
	}
	if _, err := c.request("GET", "/api/sys/auth/profile", nil, first.AccessToken, 200, 0); err != nil {
		return err
	}
	if _, err := c.request("PUT", "/api/sys/auth/password", map[string]string{"oldPassword": os.Getenv("STARS_SEED_ADMIN_PASSWORD"), "newPassword": newPassword}, first.AccessToken, 200, 0); err != nil {
		return err
	}
	if _, err := c.request("GET", "/api/sys/auth/profile", nil, first.AccessToken, 401, 2000); err != nil {
		return err
	}
	if _, err := c.request("POST", "/api/sys/auth/refresh", map[string]string{"refreshToken": first.RefreshToken}, "", 401, 2007); err != nil {
		return err
	}
	pair, err := c.login(newPassword)
	if err != nil {
		return err
	}
	for _, path := range []string{"/api/sys/users", "/api/sys/roles", "/api/sys/menus/tree", "/api/sys/depts/tree"} {
		if _, err := c.request("GET", path, nil, pair.AccessToken, 200, 0); err != nil {
			return err
		}
	}
	if _, err := c.request("POST", "/api/sys/auth/logout", nil, pair.AccessToken, 200, 0); err != nil {
		return err
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("认证、首登改密与管理接口冒烟通过")
}
