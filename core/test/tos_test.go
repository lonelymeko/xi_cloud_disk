package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"cloud_disk/core/utils"
)

// TestTOSUpload 验证 TOS 上传功能。
func TestTOSUpload(t *testing.T) {
	accessKey := os.Getenv("VOLCENGINE_ACCESS_KEY_ID")
	secretKey := os.Getenv("VOLCENGINE_SECRET_ACCESS_KEY")
	endpoint := os.Getenv("VOLCENGINE_ENDPOINT")
	region := os.Getenv("VOLCENGINE_REGION")
	bucket := os.Getenv("VOLCENGINE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || endpoint == "" || bucket == "" {
		t.Skip("TOS env not set")
	}

	// 初始化 TOS 存储
	cfg := utils.StorageConfig{
		Type:            "tos",
		Region:          region,
		Endpoint:        endpoint,
		BucketName:      bucket,
		AccessKeyID:     accessKey,
		AccessKeySecret: secretKey,
	}

	storage, err := utils.NewObjectStorage(cfg)
	if err != nil {
		t.Fatalf("create TOS storage failed: %v", err)
	}
	defer storage.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试上传
	key := fmt.Sprintf("test/%d-test.txt", time.Now().Unix())
	testData := "Hello TOS!"

	err = storage.PutObject(ctx, key, bytes.NewBufferString(testData), int64(len(testData)), "text/plain")
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	t.Logf("✅ Upload successful: %s", key)

	// 测试下载
	reader, err := storage.GetObject(ctx, key)
	if err != nil {
		t.Fatalf("get object failed: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read object failed: %v", err)
	}

	if string(got) != testData {
		t.Fatalf("content mismatch: expected %q, got %q", testData, string(got))
	}
	t.Logf("✅ Download successful: %s", key)

	// 测试删除
	err = storage.DeleteObject(ctx, key)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	t.Logf("✅ Delete successful: %s", key)
}

// TestTOSPresignedURL 验证 TOS 预签名 URL 生成。
func TestTOSPresignedURL(t *testing.T) {
	accessKey := os.Getenv("VOLCENGINE_ACCESS_KEY_ID")
	secretKey := os.Getenv("VOLCENGINE_SECRET_ACCESS_KEY")
	endpoint := os.Getenv("VOLCENGINE_ENDPOINT")
	region := os.Getenv("VOLCENGINE_REGION")
	bucket := os.Getenv("VOLCENGINE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || endpoint == "" || bucket == "" {
		t.Skip("TOS env not set")
	}

	cfg := utils.StorageConfig{
		Type:            "tos",
		Region:          region,
		Endpoint:        endpoint,
		BucketName:      bucket,
		AccessKeyID:     accessKey,
		AccessKeySecret: secretKey,
	}

	storage, err := utils.NewObjectStorage(cfg)
	if err != nil {
		t.Fatalf("create TOS storage failed: %v", err)
	}
	defer storage.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 先上传一个文件
	key := fmt.Sprintf("test/%d-share.txt", time.Now().Unix())
	testData := "Shared content"

	err = storage.PutObject(ctx, key, bytes.NewBufferString(testData), int64(len(testData)), "text/plain")
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	t.Logf("📤 Uploaded: %s", key)

	// 生成预签名 URL
	url, err := storage.GetPresignedURL(ctx, key, 1*time.Hour)
	if err != nil {
		t.Fatalf("generate presigned URL failed: %v", err)
	}

	if url == "" {
		t.Fatal("empty presigned URL")
	}

	t.Logf("✅ Presigned URL generated: %s", url)

	// 清理
	_ = storage.DeleteObject(ctx, key)
}

// TestTOSStorageManager 验证存储管理器。
func TestTOSStorageManager(t *testing.T) {
	accessKey := os.Getenv("VOLCENGINE_ACCESS_KEY_ID")
	secretKey := os.Getenv("VOLCENGINE_SECRET_ACCESS_KEY")
	endpoint := os.Getenv("VOLCENGINE_ENDPOINT")
	region := os.Getenv("VOLCENGINE_REGION")
	bucket := os.Getenv("VOLCENGINE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || endpoint == "" || bucket == "" {
		t.Skip("TOS env not set")
	}

	// 初始化存储管理器
	cfg := utils.StorageConfig{
		Type:            "tos",
		Region:          region,
		Endpoint:        endpoint,
		BucketName:      bucket,
		AccessKeyID:     accessKey,
		AccessKeySecret: secretKey,
	}

	err := utils.InitStorage(cfg)
	if err != nil {
		t.Fatalf("initialize storage failed: %v", err)
	}
	t.Logf("✅ Storage initialized")

	// 获取存储实例
	storage, err := utils.GetStorage()
	if err != nil {
		t.Fatalf("get storage failed: %v", err)
	}

	if storage == nil {
		t.Fatal("storage is nil")
	}
	t.Logf("✅ Storage retrieved from manager")

	// 关闭存储
	err = utils.CloseStorage()
	if err != nil {
		t.Fatalf("close storage failed: %v", err)
	}
	t.Logf("✅ Storage closed")

}
