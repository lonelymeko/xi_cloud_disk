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

// TestTOSMultipartUpload 验证 TOS 分片上传流程。
func TestTOSMultipartUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multipart upload test with -short flag")
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	key := fmt.Sprintf("test_multipart_%d.mp4", time.Now().Unix())
	filePath := "test.mov"

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		t.Skipf("test file not found: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("open file failed: %v", err)
	}
	defer file.Close()

	fileSize := fileInfo.Size()
	partSize := int64(5 * 1024 * 1024) // 5MB per part
	var parts []utils.Part

	t.Logf("📊 File size: %.2f MB, Part size: %.2f MB",
		float64(fileSize)/(1024*1024),
		float64(partSize)/(1024*1024))

	// Step 1: Initiate multipart upload
	t.Logf("📋 Initiating multipart upload: %s", key)
	uploadID, err := storage.InitiateMultipartUpload(ctx, key, "video/mp4")
	if err != nil {
		t.Fatalf("initiate multipart upload failed: %v", err)
	}
	t.Logf("✅ Initiated, upload ID: %s", uploadID)

	defer func() {
		if err != nil {
			abortCtx, abortCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer abortCancel()
			storage.AbortMultipartUpload(abortCtx, key, uploadID)
			t.Logf("⚠️  Aborted upload: %s", uploadID)
		}
	}()

	startTime := time.Now()
	partNumber := 1

	for offset := int64(0); offset < fileSize; offset += partSize {
		currentPartSize := partSize
		if offset+partSize > fileSize {
			currentPartSize = fileSize - offset
		}

		_, seekErr := file.Seek(offset, 0)
		if seekErr != nil {
			t.Fatalf("seek failed: %v", seekErr)
		}

		partData := make([]byte, currentPartSize)
		n, readErr := io.ReadFull(file, partData)
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			t.Fatalf("read part failed: %v", readErr)
		}

		partCtx, partCancel := context.WithTimeout(ctx, 2*time.Minute)

		partStartTime := time.Now()
		etag, uploadErr := storage.UploadPart(partCtx, key, uploadID, partNumber, bytes.NewReader(partData[:n]), int64(n))
		partCancel()

		if uploadErr != nil {
			t.Fatalf("upload part %d failed: %v", partNumber, uploadErr)
		}

		partDuration := time.Since(partStartTime)
		speed := float64(n) / partDuration.Seconds() / (1024 * 1024)

		t.Logf("📤 Part %d: %.2f MB, ETag: %s, Time: %v, Speed: %.2f MB/s",
			partNumber,
			float64(n)/(1024*1024),
			etag,
			partDuration.Round(time.Millisecond),
			speed)

		parts = append(parts, utils.Part{
			PartNumber: partNumber,
			ETag:       etag,
		})

		partNumber++
	}

	totalDuration := time.Since(startTime)
	avgSpeed := float64(fileSize) / totalDuration.Seconds() / (1024 * 1024)
	t.Logf("✅ All parts uploaded, Total time: %v, Avg speed: %.2f MB/s",
		totalDuration.Round(time.Millisecond),
		avgSpeed)

	// Step 3: Complete multipart upload
	completeCtx, completeCancel := context.WithTimeout(ctx, 1*time.Minute)
	defer completeCancel()

	err = storage.CompleteMultipartUpload(completeCtx, key, uploadID, parts)
	if err != nil {
		t.Fatalf("complete multipart upload failed: %v", err)
	}

	t.Logf("🎉 Multipart upload completed!")
	t.Logf("   Bucket: %s", bucket)
	t.Logf("   Key: %s", key)
	t.Logf("   Parts: %d", len(parts))
}

// TestTOSMultipartCancel 验证分片上传取消。
func TestTOSMultipartCancel(t *testing.T) {
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

	key := fmt.Sprintf("test_cancel_%d.txt", time.Now().Unix())

	uploadID, err := storage.InitiateMultipartUpload(ctx, key, "text/plain")
	if err != nil {
		t.Fatalf("initiate failed: %v", err)
	}
	t.Logf("✅ Initiated, ID: %s", uploadID)

	err = storage.AbortMultipartUpload(ctx, key, uploadID)
	if err != nil {
		t.Fatalf("abort failed: %v", err)
	}
	t.Logf("✅ Aborted successfully")
}

// TestTOSStorageInterface 验证 TOS 存储接口完整性。
func TestTOSStorageInterface(t *testing.T) {
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
		t.Fatalf("create storage failed: %v", err)
	}
	defer storage.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testKey := fmt.Sprintf("test_interface_%d.txt", time.Now().Unix())
	testData := "Interface test data"

	// Test PutObject
	t.Log("🧪 Testing PutObject...")
	err = storage.PutObject(ctx, testKey, bytes.NewBufferString(testData), int64(len(testData)), "text/plain")
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}
	t.Logf("✅ PutObject passed")

	// Test GetObject
	t.Log("🧪 Testing GetObject...")
	reader, err := storage.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(got) != testData {
		t.Fatalf("content mismatch")
	}
	t.Logf("✅ GetObject passed")

	// Test GetPresignedURL
	t.Log("🧪 Testing GetPresignedURL...")
	url, err := storage.GetPresignedURL(ctx, testKey, 1*time.Hour)
	if err != nil {
		t.Fatalf("GetPresignedURL failed: %v", err)
	}
	if url == "" {
		t.Fatal("empty URL")
	}
	t.Logf("✅ GetPresignedURL passed")

	// Test DeleteObject
	t.Log("🧪 Testing DeleteObject...")
	err = storage.DeleteObject(ctx, testKey)
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}
	t.Logf("✅ DeleteObject passed")

	t.Logf("\n✨ All interface tests passed!")
}
