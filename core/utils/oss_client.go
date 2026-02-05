package utils

import (
	"os"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// newOSSClient 创建 OSS 客户端。
func newOSSClient(region string) (*oss.Client, error) {
	if err := ossLoadEnv(); err != nil {
		return nil, err
	}
	provider := credentials.NewEnvironmentVariableCredentialsProvider()
	if os.Getenv("OSS_ACCESS_KEY_ID") == "" || os.Getenv("OSS_ACCESS_KEY_SECRET") == "" {
		provider = credentials.NewAnonymousCredentialsProvider()
	}
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(provider).
		WithRegion(region)
	return oss.NewClient(cfg), nil
}
