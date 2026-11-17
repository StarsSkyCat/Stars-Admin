package captcha

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestCaptchaImageAndRandomAnswer(t *testing.T) {
	a, image, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 6 {
		t.Fatal("验证码长度错误")
	}
	for _, digit := range a {
		if digit < '0' || digit > '9' {
			t.Fatal("验证码包含非数字")
		}
	}
	_, data, ok := strings.Cut(image, ",")
	if !ok {
		t.Fatal("不是 data URL")
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil || len(decoded) < 100 || string(decoded[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("验证码图片无效")
	}
}
