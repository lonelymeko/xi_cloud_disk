package utils

import (
	"context"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

// PresignGetObject 生成获取对象的临时签名 URL。
func PresignGetObject(ctx context.Context, objectKey string, expires time.Duration) (string, error) {
	if err := ossLoadEnv(); err != nil {
		return "", err
	}
	client, err := newOSSClient(OSSRegionValue())
	if err != nil {
		return "", err
	}
	result, err := client.Presign(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(OSSBucketNameValue()),
		Key:    oss.Ptr(objectKey),
	}, oss.PresignExpires(expires))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}
