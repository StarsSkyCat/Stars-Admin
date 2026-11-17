// Package jwt 提供限定算法、签发方和用途的双令牌。
package jwt

import (
	"errors"
	"strconv"
	"time"
	"uuid"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// Claims 是令牌声明；用户 ID 用十进制字符串表达。
type Claims struct {
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	SID       string `json:"sid"`
	TokenType string `json:"tokenType"`
	jwtlib.RegisteredClaims
}

// Tokens 是候选双令牌，只有会话原子写入成功后才能返回。
type Tokens struct {
	AccessToken        string `json:"accessToken"`
	RefreshToken       string `json:"refreshToken"`
	TokenType          string `json:"tokenType"`
	ExpiresIn          int64  `json:"expiresIn"`
	MustChangePassword bool   `json:"mustChangePassword"`
	Access             Claims `json:"-"`
	Refresh            Claims `json:"-"`
}

// Manager 将密钥与允许的签发参数封装在实例内。
type Manager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
}

// New 创建签发器，配置校验在应用初始化执行。
func New(secret, issuer string, accessTTL time.Duration) *Manager {
	return &Manager{[]byte(secret), issuer, accessTTL}
}

// Issue 令牌轮换保持原始会话的绝对期限。
func (m *Manager) Issue(id int64, username, sid string, deadline time.Time, mustChange bool) (Tokens, error) {
	now := time.Now().UTC().Truncate(time.Second)
	deadline = deadline.UTC().Truncate(time.Second)
	if id <= 0 || username == "" || !deadline.After(now) {
		return Tokens{}, errors.New("令牌签发参数无效")
	}
	if sid == "" {
		sid = uuid.New().String()
	}
	accessEnd := now.Add(m.accessTTL)
	if accessEnd.After(deadline) {
		accessEnd = deadline
	}
	makeClaims := func(kind string, end time.Time) Claims {
		return Claims{UserID: strconv.FormatInt(id, 10), Username: username, SID: sid, TokenType: kind,
			Issuer: m.issuer, IssuedAt: jwtlib.NewNumericDate(now), ExpiresAt: jwtlib.NewNumericDate(end), ID: uuid.New().String()}
	}
	result := Tokens{TokenType: "Bearer", ExpiresIn: accessEnd.Unix() - now.Unix(), MustChangePassword: mustChange,
		Access: makeClaims("access", accessEnd), Refresh: makeClaims("refresh", deadline)}
	var err error
	result.AccessToken, err = jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, result.Access).SignedString(m.secret)
	if err != nil {
		return Tokens{}, err
	}
	result.RefreshToken, err = jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, result.Refresh).SignedString(m.secret)
	return result, err
}

// Parse 拒绝算法降级、用途混淆以及缺失的必需声明。
func (m *Manager) Parse(raw, kind string) (*Claims, error) {
	claims := new(Claims)
	_, err := jwtlib.ParseWithClaims(raw, claims, func(_ *jwtlib.Token) (any, error) { return m.secret, nil },
		jwtlib.WithValidMethods([]string{"HS256"}), jwtlib.WithIssuer(m.issuer), jwtlib.WithExpirationRequired(), jwtlib.WithIssuedAt())
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != claims.UserID || claims.Username == "" || claims.TokenType != kind || claims.IssuedAt == nil || claims.ExpiresAt == nil || !claims.ExpiresAt.After(claims.IssuedAt.Time) {
		return nil, errors.New("令牌声明无效")
	}
	if _, err := uuid.Parse(claims.SID); err != nil {
		return nil, errors.New("会话标识无效")
	}
	if _, err := uuid.Parse(claims.ID); err != nil {
		return nil, errors.New("令牌标识无效")
	}
	return claims, nil
}

// Expired 只用于区分已经通过解析流程发现的过期错误。
func Expired(err error) bool { return errors.Is(err, jwtlib.ErrTokenExpired) }
