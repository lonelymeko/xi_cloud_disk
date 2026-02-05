package test

import (
	"bytes"
	"cloud_disk/core/common"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/joho/godotenv"
)

// TestInitiateMultipartUpload 验证分片上传流程。
func TestInitiateMultipartUpload(t *testing.T) {
	// 设置更长的测试超时时间（5分钟）
	if testing.Short() {
		t.Skip("跳过分片上传测试（使用 -short 标志）")
	}

	// 从环境变量获取访问凭证
	err := godotenv.Load("../.env")
	if err != nil {
		t.Fatalf("加载 .env 文件失败: %v", err)
	}

	// 配置OSS客户端，设置凭证提供者和Endpoint
	config := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(common.OSSRegion)

	// 初始化OSS客户端
	client := oss.NewClient(config)

	// 配置Bucket和文件信息
	bucket := common.OSSBucketName
	key := fmt.Sprintf("test_multipart_%d.mov", time.Now().Unix())
	filePath := "test.mov"

	// 检查文件是否存在
	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		t.Skipf("测试文件不存在: %s", filePath)
	}

	// 步骤1：初始化分片上传
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	initResult, err := client.InitiateMultipartUpload(ctx, &oss.InitiateMultipartUploadRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(key),
	})
	if err != nil {
		t.Fatalf("初始化分片上传失败: %v", err)
	}

	uploadId := *initResult.UploadId
	t.Logf("✅ 初始化分片上传成功，上传ID: %s", uploadId)

	// 确保失败时取消上传
	defer func() {
		if err != nil {
			abortCtx, abortCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer abortCancel()
			client.AbortMultipartUpload(abortCtx, &oss.AbortMultipartUploadRequest{
				Bucket:   oss.Ptr(bucket),
				Key:      oss.Ptr(key),
				UploadId: oss.Ptr(uploadId),
			})
			t.Logf("⚠️  已取消上传任务: %s", uploadId)
		}
	}()

	// 步骤2：上传分片
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("打开文件失败: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}

	fileSize := fileInfo.Size()
	partSize := int64(5 * 1024 * 1024) // 每个分片 5MB（推荐最小值）
	partNumber := int32(1)
	var parts []oss.UploadPart

	t.Logf("📊 文件大小: %.2f MB, 分片大小: %.2f MB",
		float64(fileSize)/(1024*1024),
		float64(partSize)/(1024*1024))

	startTime := time.Now()

	for offset := int64(0); offset < fileSize; offset += partSize {
		// 计算当前分片大小
		currentPartSize := partSize
		if offset+partSize > fileSize {
			currentPartSize = fileSize - offset
		}

		// 创建分片数据缓冲区
		partData := make([]byte, currentPartSize)

		// 定位到分片起始位置并读取数据
		_, seekErr := file.Seek(offset, 0)
		if seekErr != nil {
			t.Fatalf("文件定位失败: %v", seekErr)
		}

		n, readErr := io.ReadFull(file, partData)
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			t.Fatalf("读取文件分片失败: %v", readErr)
		}

		// 为每个分片设置独立的超时上下文（2分钟）
		partCtx, partCancel := context.WithTimeout(ctx, 2*time.Minute)

		// 上传分片（使用 bytes.NewReader）
		partStartTime := time.Now()
		partResult, uploadErr := client.UploadPart(partCtx, &oss.UploadPartRequest{
			Bucket:     oss.Ptr(bucket),
			Key:        oss.Ptr(key),
			UploadId:   oss.Ptr(uploadId),
			PartNumber: partNumber,
			Body:       bytes.NewReader(partData[:n]),
		})
		partCancel()

		if uploadErr != nil {
			t.Fatalf("上传分片 %d 失败: %v", partNumber, uploadErr)
		}

		partDuration := time.Since(partStartTime)
		speed := float64(n) / partDuration.Seconds() / (1024 * 1024) // MB/s

		t.Logf("📤 分片 %d: %.2f MB, ETag: %s, 耗时: %v, 速度: %.2f MB/s",
			partNumber,
			float64(n)/(1024*1024),
			*partResult.ETag,
			partDuration.Round(time.Millisecond),
			speed)

		// 记录已上传的分片信息
		parts = append(parts, oss.UploadPart{
			PartNumber: partNumber,
			ETag:       partResult.ETag,
		})

		partNumber++
	}

	totalDuration := time.Since(startTime)
	avgSpeed := float64(fileSize) / totalDuration.Seconds() / (1024 * 1024)
	t.Logf("✅ 所有分片上传完成，总耗时: %v, 平均速度: %.2f MB/s",
		totalDuration.Round(time.Millisecond),
		avgSpeed)

	// 步骤3：完成分片上传
	completeCtx, completeCancel := context.WithTimeout(ctx, 1*time.Minute)
	defer completeCancel()

	completeResult, err := client.CompleteMultipartUpload(completeCtx, &oss.CompleteMultipartUploadRequest{
		Bucket:   oss.Ptr(bucket),
		Key:      oss.Ptr(key),
		UploadId: oss.Ptr(uploadId),
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		t.Fatalf("完成分片上传失败: %v", err)
	}

	t.Logf("🎉 分片上传完成！")
	t.Logf("   Bucket: %s", *completeResult.Bucket)
	t.Logf("   Key: %s", *completeResult.Key)
	t.Logf("   Location: %s", *completeResult.Location)
	t.Logf("   ETag: %s", *completeResult.ETag)
}
