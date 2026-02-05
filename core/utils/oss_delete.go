package utils

import (
	"context"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

// DeleteOSSObject 删除 OSS 对象。
func DeleteOSSObject(ctx context.Context, objectKey string) error {
	if err := ossLoadEnv(); err != nil {
		return err
	}
	client, err := newOSSClient(OSSRegionValue())
	if err != nil {
		return err
	}
	_, err = client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(OSSBucketNameValue()),
		Key:    oss.Ptr(objectKey),
	})
	return err
}
