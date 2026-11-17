// Package hash 提供带长度边界的 bcrypt 密码哈希。
package hash

import (
	"errors"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// Validate 拒绝过短或会被 bcrypt 截断的密码。
func Validate(password string) error {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 12 || len(password) > 72 {
		return errors.New("密码至少 12 个字符且不能超过 72 字节")
	}
	return nil
}

// Password 校验后生成哈希，禁止调用方记录明文。
func Password(password string, cost int) (string, error) {
	if err := Validate(password); err != nil {
		return "", err
	}
	data, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(data), err
}

// Match 比较密码，错误与未知用户在认证层统一处理。
func Match(encoded, password string) bool {
	return len(password) <= 72 && bcrypt.CompareHashAndPassword([]byte(encoded), []byte(password)) == nil
}
