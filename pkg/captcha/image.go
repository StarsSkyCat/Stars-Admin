// Package captcha 仅负责安全随机题目与图像，不隐藏持久化错误。
package captcha

import (
	"crypto/rand"
	"image/color"
	"math/big"
	"strings"

	"github.com/mojocn/base64Captcha"
)

// Generate 返回六位数字答案及 PNG data URL；答案只交给服务端存储。
func Generate() (answer, image string, err error) {
	var b strings.Builder
	for range 6 {
		digit, e := rand.Int(rand.Reader, big.NewInt(10))
		if e != nil {
			return "", "", e
		}
		b.WriteByte('0' + byte(digit.Int64()))
	}
	answer = b.String()
	driver := base64Captcha.NewDriverDigit(80, 240, 6, 0.3, 24)
	item, err := driver.DrawCaptcha(answer)
	if err != nil {
		return "", "", err
	}
	// 数字渲染器使用索引 0 作背景、索引 1 作数字及曲线，固定对比度兼容深色主题。
	palette := item.(*base64Captcha.ItemDigit).Palette
	palette[0] = color.RGBA{R: 246, G: 249, B: 253, A: 255}
	palette[1] = color.RGBA{R: 20, G: 35, B: 57, A: 255}
	noise := palette[2:]
	for i := range noise {
		noise[i] = color.RGBA{R: 175, G: 195, B: 218, A: 255}
	}
	return answer, item.EncodeB64string(), nil
}
