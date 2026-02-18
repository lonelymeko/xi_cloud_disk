package utils

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/joho/godotenv"
)

// ossLoadEnv 加载 OSS 相关环境变量。
var ossLoadEnv = func() error { return godotenv.Load(".env") }

// ossKeyGen 生成 OSS 对象键。
var ossKeyGen = func(originalFilename string) string { return UUID() + path.Ext(originalFilename) }

// ossUpload 执行 OSS 上传。
var ossUpload = func(region, bucket, key string, body io.Reader) (string, error) {
	client, err := newOSSClient(region)
	if err != nil {
		return "", err
	}
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

// SetOSSLoadEnv 设置环境变量加载函数。
func SetOSSLoadEnv(loader func() error) {
	ossLoadEnv = loader
}

// SetOSSKeyGen 设置对象键生成函数。
func SetOSSKeyGen(keyGen func(originalFilename string) string) {
	ossKeyGen = keyGen
}

// SetOSSUpload 设置上传函数。
func SetOSSUpload(upload func(region, bucket, key string, body io.Reader) (string, error)) {
	ossUpload = upload
}

// OSSLoadEnv 返回当前环境变量加载函数。
func OSSLoadEnv() func() error {
	return ossLoadEnv
}

// OSSKeyGen 返回当前对象键生成函数。
func OSSKeyGen() func(originalFilename string) string {
	return ossKeyGen
}

// OSSUpload 返回当前上传函数。
func OSSUpload() func(region, bucket, key string, body io.Reader) (string, error) {
	return ossUpload
}

// UploadToOSS 上传文件到 OSS。
func UploadToOSS(fileReader io.Reader, originalFilename string) (string, error) {
	key := ossKeyGen(originalFilename)
	if err := ossLoadEnv(); err != nil {
		return "", fmt.Errorf("failed to load env: %w", err)
	}
	var (
		region     = OSSRegionValue()
		bucketName = OSSBucketNameValue()
		objectName = key
	)
	etag, err := ossUpload(region, bucketName, objectName, fileReader)
	if err != nil {
		return "", fmt.Errorf("failed to put object: %w", err)
	}

	fmt.Printf("put object sucessfully, ETag :%v\n", etag)
	return objectName, nil
}
