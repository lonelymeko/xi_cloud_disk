package utils

import (
	"cloud_disk/core/common"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// OSSHost 获取 OSS 连接地址。
func OSSHost() string {
	if v := os.Getenv("OSS_HOST"); v != "" {
		return v
	}
	bucket := OSSBucketNameValue()
	region := OSSRegionValue()
	if region != "" && !strings.HasPrefix(region, "oss-") {
		region = "oss-" + region
	}
	return fmt.Sprintf("%s.%s.aliyuncs.com:443", bucket, region)
}

func OSSRegionValue() string {
	if v := os.Getenv("OSS_REGION"); v != "" {
		return normalizeOSSRegion(v)
	}
	return normalizeOSSRegion(common.OSSRegion)
}

func normalizeOSSRegion(region string) string {
	r := strings.TrimSpace(region)
	for strings.HasPrefix(r, "oss-") {
		r = strings.TrimPrefix(r, "oss-")
	}
	return r
}

func OSSBucketNameValue() string {
	if v := os.Getenv("OSS_BUCKET_NAME"); v != "" {
		return v
	}
	return common.OSSBucketName
}

// OSSConnectivity 检查 OSS 网络连通性。
func OSSConnectivity(ctx context.Context) error {
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", OSSHost())
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
