package utils

import (
	"os"
	"strconv"
	"time"
)

// defaultRecycleDays 默认回收周期（天）。
const defaultRecycleDays = 30

// RecycleTTL 获取回收等待时长。
func RecycleTTL() time.Duration {
	if v := os.Getenv("RECYCLE_TTL_SECONDS"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	if v := os.Getenv("RECYCLE_TTL_DAYS"); v != "" {
		if days, err := strconv.Atoi(v); err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	return time.Duration(defaultRecycleDays) * 24 * time.Hour
}

// RecycleScanInterval 获取回收扫描间隔。
func RecycleScanInterval() time.Duration {
	if v := os.Getenv("RECYCLE_SCAN_INTERVAL_SECONDS"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return time.Hour
}
