package utils

import (
	"cloud_disk/core/common"
	"context"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

// DeleteOSSObject 删除 OSS 对象。
func DeleteOSSObject(ctx context.Context, objectKey string) error {
	client, err := newOSSClient(common.OSSRegion)
	if err != nil {
		return err
	}
	_, err = client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(common.OSSBucketName),
		Key:    oss.Ptr(objectKey),
	})
	return err
}
