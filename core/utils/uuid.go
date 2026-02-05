package utils

import (
	"github.com/google/uuid"
)

// UUID 生成 UUID 字符串。
func UUID() string {
	return uuid.New().String()
}
