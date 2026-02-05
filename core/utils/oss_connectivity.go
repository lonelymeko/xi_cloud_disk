package utils

import (
	"cloud_disk/core/common"
	"context"
	"fmt"
	"net"
	"os"
	"time"
)

// OSSHost 获取 OSS 连接地址。
func OSSHost() string {
	if v := os.Getenv("OSS_HOST"); v != "" {
		return v
	}
	bucket := os.Getenv("OSS_BUCKET_NAME")
	region := os.Getenv("OSS_REGION")
	if bucket == "" {
		bucket = common.OSSBucketName
	}
	if region == "" {
		region = common.OSSRegion
	}
	return fmt.Sprintf("%s.%s.aliyuncs.com:443", bucket, region)
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
