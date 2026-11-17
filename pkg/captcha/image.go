// Package captcha 仅负责安全随机题目与图像，不隐藏持久化错误。
package captcha

import (
	"crypto/rand"
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
	driver := base64Captcha.NewDriverDigit(60, 180, 6, 0.7, 80)
	item, err := driver.DrawCaptcha(answer)
	if err != nil {
		return "", "", err
	}
	return answer, item.EncodeB64string(), nil
}
