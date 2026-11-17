package jwt

import (
	"strings"
	"testing"
	"time"
	"uuid"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestIssueAndStrictClaims(t *testing.T) {
	secret := strings.Repeat("s", 32)
	m := New(secret, "test", time.Minute)
	deadline := time.Now().Add(time.Hour).Truncate(time.Second)
	pair, err := m.Issue(9223372036854775807, "alice", "", deadline, true)
	if err != nil {
		t.Fatal(err)
	}
	a, err := m.Parse(pair.AccessToken, "access")
	if err != nil || a.UserID != "9223372036854775807" || a.SID != pair.Refresh.SID || a.ID == pair.Refresh.ID || !pair.MustChangePassword {
		t.Fatal("令牌声明错误")
	}
	if _, err := m.Parse(pair.RefreshToken, "access"); err == nil {
		t.Fatal("Refresh 不能用于 Access")
	}
	rotated, err := m.Issue(1, "alice", pair.Refresh.SID, deadline, true)
	if err != nil || !rotated.Refresh.ExpiresAt.Equal(deadline) {
		t.Fatal("刷新改变绝对期限")
	}
	valid := Claims{UserID: "1", Username: "alice", SID: uuid.New().String(), TokenType: "access", Issuer: "test", IssuedAt: jwtlib.NewNumericDate(time.Now().Add(-time.Minute)), ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Minute)), ID: uuid.New().String()}
	for _, name := range []string{"issuer", "issuedAt", "expiry", "sid", "jti", "id", "future", "purpose", "expired"} {
		t.Run(name, func(t *testing.T) {
			c := valid
			switch name {
			case "issuer":
				c.Issuer = "other"
			case "issuedAt":
				c.IssuedAt = nil
			case "expiry":
				c.ExpiresAt = nil
			case "sid":
				c.SID = "invalid"
			case "jti":
				c.ID = "invalid"
			case "id":
				c.UserID = "01"
			case "future":
				c.IssuedAt = jwtlib.NewNumericDate(time.Now().Add(time.Hour))
			case "purpose":
				c.TokenType = "refresh"
			case "expired":
				c.ExpiresAt = jwtlib.NewNumericDate(time.Now().Add(-time.Second))
			}
			raw, e := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, c).SignedString([]byte(secret))
			if e != nil {
				t.Fatal(e)
			}
			if _, err := m.Parse(raw, "access"); err == nil {
				t.Fatal("无效声明未拒绝")
			}
		})
	}
	for _, method := range []jwtlib.SigningMethod{jwtlib.SigningMethodHS384, jwtlib.SigningMethodNone} {
		var key any = []byte(secret)
		if method == jwtlib.SigningMethodNone {
			key = jwtlib.UnsafeAllowNoneSignatureType
		}
		raw, e := jwtlib.NewWithClaims(method, valid).SignedString(key)
		if e != nil {
			t.Fatal(e)
		}
		if _, e := m.Parse(raw, "access"); e == nil {
			t.Fatal("算法白名单未生效")
		}
	}
	if _, err := New("other-secret", "test", time.Minute).Parse(pair.AccessToken, "access"); err == nil {
		t.Fatal("错误签名未拒绝")
	}
}
