package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/bcrypt"
)

// JWTClaims JWT声明结构
type JWTClaims struct {
	UserID      uint     `json:"user_id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

var (
	// JWT密钥，应该从配置文件中读取
	jwtSecret = []byte("your-secret-key-here")
)

// GenerateJWT 生成JWT token
func GenerateJWT(userID uint, username string, roles []string, permissions []string) (string, error) {
	claims := &JWTClaims{
		UserID:      userID,
		Username:    username,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateJWT 验证JWT token
func ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// HashPassword 密码哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetTokenHash 获取token哈希值
func GetTokenHash(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}

// BlacklistToken 将token加入黑名单
func BlacklistToken(rdb *redis.Client, token string, expiration time.Duration) error {
	ctx := context.Background()
	tokenHash := GetTokenHash(token)
	return rdb.Set(ctx, fmt.Sprintf("blacklist:%s", tokenHash), "1", expiration).Err()
}

// IsTokenBlacklisted 检查token是否在黑名单中
func IsTokenBlacklisted(rdb *redis.Client, token string) bool {
	ctx := context.Background()
	tokenHash := GetTokenHash(token)
	result := rdb.Get(ctx, fmt.Sprintf("blacklist:%s", tokenHash))
	return result.Err() == nil
}

// GenerateRefreshToken 生成刷新token
func GenerateRefreshToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// StoreRefreshToken 存储刷新token
func StoreRefreshToken(rdb *redis.Client, userID uint, refreshToken string, expiration time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("refresh_token:%d", userID)
	return rdb.Set(ctx, key, refreshToken, expiration).Err()
}

// ValidateRefreshToken 验证刷新token
func ValidateRefreshToken(rdb *redis.Client, userID uint, refreshToken string) bool {
	ctx := context.Background()
	key := fmt.Sprintf("refresh_token:%d", userID)
	result := rdb.Get(ctx, key)
	if result.Err() != nil {
		return false
	}
	return result.Val() == refreshToken
}

// DeleteRefreshToken 删除刷新token
func DeleteRefreshToken(rdb *redis.Client, userID uint) error {
	ctx := context.Background()
	key := fmt.Sprintf("refresh_token:%d", userID)
	return rdb.Del(ctx, key).Err()
}