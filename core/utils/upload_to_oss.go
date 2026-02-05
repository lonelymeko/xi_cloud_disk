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

var ossLoadEnv = func() error { return godotenv.Load(".env") }
var ossKeyGen = func(originalFilename string) string { return UUID() + path.Ext(originalFilename) }
var ossUpload = func(region, bucket, key string, body io.Reader) (string, error) {
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)
	client := oss.NewClient(cfg)
	result, err := client.PutObject(context.TODO(), &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(key),
		Body:   body,
	})
	if err != nil {
		return "", err
	}
	return oss.ToString(result.ETag), nil
}

func SetOSSLoadEnv(loader func() error) {
	ossLoadEnv = loader
}

func SetOSSKeyGen(keyGen func(originalFilename string) string) {
	ossKeyGen = keyGen
}

func SetOSSUpload(upload func(region, bucket, key string, body io.Reader) (string, error)) {
	ossUpload = upload
}

func OSSLoadEnv() func() error {
	return ossLoadEnv
}

func OSSKeyGen() func(originalFilename string) string {
	return ossKeyGen
}

func OSSUpload() func(region, bucket, key string, body io.Reader) (string, error) {
	return ossUpload
}

// UploadToOSS 上传文件到 OSS，接受 io.Reader 和原始文件名
func UploadToOSS(fileReader io.Reader, originalFilename string) (string, error) {
	key := ossKeyGen(originalFilename)

	var (
		region     = common.OSSRegion
		bucketName = common.OSSBucketName
		objectName = key
	)
	if err := ossLoadEnv(); err != nil {
		panic(err)
	}
	etag, err := ossUpload(region, bucketName, objectName, fileReader)
	if err != nil {
		return "", fmt.Errorf("failed to put object: %w", err)
	}

	fmt.Printf("put object sucessfully, ETag :%v\n", etag)
	return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", bucketName, region, objectName), nil
}
