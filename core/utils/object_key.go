package utils

import (
	"net/url"
	"strings"
)

// ObjectKeyFromPath 从路径或 URL 中提取对象键。
func ObjectKeyFromPath(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Path != "" {
		return strings.TrimPrefix(parsed.Path, "/")
	}
	idx := strings.Index(raw, "/")
	if idx == -1 {
		return ""
	}
	return strings.TrimPrefix(raw[idx:], "/")
}
