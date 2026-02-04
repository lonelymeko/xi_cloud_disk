package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/joho/godotenv"
)

func TestOSS(t *testing.T) {
	var (
		region     = "cn-beijing"
		bucketName = "xi-cloud-disk"
		objectName = "test.txt"
		localFile  = "/Users/xixiu/GolandProjects/cloud_disk/core/test/test.txt"
	)
	err := godotenv.Load("../.env")
	if err != nil {
		panic(err)
	}
	// Using the SDK's default configuration
	// loading credentials values from the environment variables
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	client := oss.NewClient(cfg)

	file, err := os.Open(localFile)
	if err != nil {
		log.Fatalf("failed to open file %v", err)
	}
	defer file.Close()

	result, err := client.PutObject(context.TODO(), &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucketName),
		Key:    oss.Ptr(objectName),
		Body:   file,
	})

	if err != nil {
		log.Fatalf("failed to put object %v", err)
	}

	fmt.Printf("put object sucessfully, ETag :%v\n", result.ETag)
}
