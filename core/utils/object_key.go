package utils

import (
	"net/url"
	"strings"
)

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
