package utils

import (
	"cloud_disk/core/common"
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/joho/godotenv"
)

// UploadToOSS 上传文件到 OSS，接受 io.Reader 和原始文件名
func UploadToOSS(fileReader io.Reader, originalFilename string) (string, error) {
	key := UUID() + path.Ext(originalFilename)

	var (
		region     = common.OSSRegion
		bucketName = common.OSSBucketName
		objectName = key
	)
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	// Using the SDK's default configuration
	// loading credentials values from the environment variables
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	client := oss.NewClient(cfg)

	// 同步上传，不使用 goroutine
	result, err := client.PutObject(context.TODO(), &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucketName),
		Key:    oss.Ptr(objectName),
		Body:   fileReader,
	})
	if err != nil {
		return "", fmt.Errorf("failed to put object: %w", err)
	}

	fmt.Printf("put object sucessfully, ETag :%v\n", result.ETag)
	return fmt.Sprintf("https://%s.%s.aliyuncs.com/%s", bucketName, region, objectName), nil
}
