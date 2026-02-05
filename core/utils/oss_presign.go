package utils

import (
	"cloud_disk/core/common"
	"context"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

func PresignGetObject(ctx context.Context, objectKey string, expires time.Duration) (string, error) {
	client, err := newOSSClient(common.OSSRegion)
	if err != nil {
		return "", err
	}
	result, err := client.Presign(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(common.OSSBucketName),
		Key:    oss.Ptr(objectKey),
	}, oss.PresignExpires(expires))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}
