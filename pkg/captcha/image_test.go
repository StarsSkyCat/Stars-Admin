package captcha

import (
	"bytes"
	"encoding/base64"
	"image/png"
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
	rendered, err := png.Decode(bytes.NewReader(decoded))
	if err != nil {
		t.Fatal(err)
	}
	darkPixels := 0
	for y := range rendered.Bounds().Dy() {
		for x := range rendered.Bounds().Dx() {
			r, g, b, a := rendered.At(x, y).RGBA()
			if a != 0xffff {
				t.Fatal("验证码底色应保持不透明")
			}
			if r < 0x4000 && g < 0x4000 && b < 0x4000 {
				darkPixels++
			}
		}
	}
	if darkPixels < 200 {
		t.Fatal("验证码缺少足够的高对比度笔画")
	}
}
