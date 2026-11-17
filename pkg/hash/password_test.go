package hash

import (
	"strings"
	"testing"
)

func TestPasswordBoundariesAndComparison(t *testing.T) {
	for _, password := range []string{"short", strings.Repeat("a", 73), strings.Repeat("中", 25), string([]byte{0xff, 0xfe})} {
		if _, err := Password(password, 4); err == nil {
			t.Fatal("密码边界未拒绝")
		}
	}
	for _, password := range []string{"Correct-Horse-123", strings.Repeat("中", 24), strings.Repeat("a", 72)} {
		encoded, err := Password(password, 4)
		if err != nil || encoded == password || !Match(encoded, password) || Match(encoded, "wrong-password") {
			t.Fatal("哈希或密码比较不正确")
		}
	}
}
