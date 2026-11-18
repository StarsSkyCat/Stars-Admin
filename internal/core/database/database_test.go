package database

import (
	"net/url"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
)

func TestDSNEscapesCredentialsAndSetsUTC(t *testing.T) {
	c := config.Database{Host: "::1", Port: 5432, Username: "user@name", Password: "p@ss:#word", DBName: "stars_admin", SSLMode: "disable", Timeout: time.Second}
	u, err := url.Parse(DSN(c))
	if err != nil {
		t.Fatal(err)
	}
	password, _ := u.User.Password()
	if u.User.Username() != c.Username || password != c.Password || u.Host != "[::1]:5432" || u.Query().Get("TimeZone") != "UTC" {
		t.Fatal("连接串转义或时区错误")
	}
	version, err := LatestVersion()
	if err != nil || version != 6 {
		t.Fatalf("内嵌迁移最新版本错误：%d，%v", version, err)
	}
}
