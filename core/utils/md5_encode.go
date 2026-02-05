package utils

import (
	"crypto/md5"
	"fmt"
)

// Md5 计算字符串的 MD5 值。
func Md5(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
