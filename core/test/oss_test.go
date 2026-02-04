package test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

func main() {
	var (
		region     = "cn-beijing"
		bucketName = "xi-cloud-disk"
		objectName = "your object name"
		localFile  = "your local file path"
	)

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
