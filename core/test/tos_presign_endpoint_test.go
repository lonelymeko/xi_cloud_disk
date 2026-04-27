package test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"cloud_disk/core/utils"
)

func TestTOSPresignedURLUsesPresignEndpoint(t *testing.T) {
	storage, err := utils.NewObjectStorage(utils.StorageConfig{
		Type:            "tos",
		Region:          "cn-beijing",
		Endpoint:        "tos-cn-beijing.ivolces.com",
		PresignEndpoint: "tos-cn-beijing.volces.com",
		BucketName:      "test-bucket",
		AccessKeyID:     "test-ak",
		AccessKeySecret: "test-sk",
	})
	if err != nil {
		t.Fatalf("create TOS storage failed: %v", err)
	}
	defer storage.Close()

	signedURL, err := storage.GetPresignedURL(context.Background(), "test/file.txt", time.Hour)
	if err != nil {
		t.Fatalf("generate presigned URL failed: %v", err)
	}

	parsed, err := url.Parse(signedURL)
	if err != nil {
		t.Fatalf("parse presigned URL failed: %v", err)
	}

	if parsed.Host != "test-bucket.tos-cn-beijing.volces.com" {
		t.Fatalf("unexpected presigned URL host: %s", parsed.Host)
	}
}
