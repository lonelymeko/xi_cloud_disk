package utils

import (
	"bytes"
	"cloud_disk/core/common"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/zeromicro/go-zero/core/logx"
)

// UploadToOSSMultipart 分片上传文件到 OSS。
// filePath: 本地文件路径。
// originalFilename: 原始文件名。
// fileSize: 文件大小（字节）。
func UploadToOSSMultipart(filePath string, originalFilename string, fileSize int64) (string, error) {
	key := UUID() + path.Ext(originalFilename)

	if err := ossLoadEnv(); err != nil {
		return "", fmt.Errorf("failed to load env: %w", err)
	}
	var (
		region     = OSSRegionValue()
		bucketName = OSSBucketNameValue()
		objectName = key
	)

	client, err := newOSSClient(region)
	if err != nil {
		return "", err
	}

	// 设置总超时时间（根据文件大小动态计算，最少 5 分钟）
	timeout := time.Duration(fileSize/1024/1024) * time.Second * 2 // 每 MB 2 秒
	if timeout < 5*time.Minute {
		timeout = 5 * time.Minute
	}
	if timeout > 30*time.Minute {
		timeout = 30 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 步骤 1: 初始化分片上传
	logx.Infof("开始分片上传: 文件大小 %.2f MB", float64(fileSize)/(1024*1024))

	initResult, err := client.InitiateMultipartUpload(ctx, &oss.InitiateMultipartUploadRequest{
		Bucket: oss.Ptr(bucketName),
		Key:    oss.Ptr(objectName),
	})
	if err != nil {
		return "", fmt.Errorf("初始化分片上传失败: %w", err)
	}

	uploadId := *initResult.UploadId
	logx.Infof("初始化分片上传成功，上传ID: %s", uploadId)

	// 确保失败时取消上传
	var uploadSuccess bool
	defer func() {
		if !uploadSuccess {
			abortCtx, abortCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer abortCancel()
			client.AbortMultipartUpload(abortCtx, &oss.AbortMultipartUploadRequest{
				Bucket:   oss.Ptr(bucketName),
				Key:      oss.Ptr(objectName),
				UploadId: oss.Ptr(uploadId),
			})
			logx.Errorf("分片上传失败，已取消上传任务: %s", uploadId)
		}
	}()

	// 步骤 2: 打开文件
	file, err := openFile(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 步骤 3: 计算分片信息
	partSize := int64(common.PartSize)
	totalParts := (fileSize + partSize - 1) / partSize // 向上取整

	logx.Infof("分片信息: 文件大小 %.2f MB, 分片大小 %.2f MB, 总分片数 %d",
		float64(fileSize)/(1024*1024),
		float64(partSize)/(1024*1024),
		totalParts)

	// 步骤 4: 并发上传分片
	parts, err := uploadPartsParallel(ctx, client, file, &uploadPartsConfig{
		Bucket:     bucketName,
		Key:        objectName,
		UploadId:   uploadId,
		FileSize:   fileSize,
		PartSize:   partSize,
		TotalParts: totalParts,
	})
	if err != nil {
		return "", err
	}

	// 步骤 5: 完成分片上传
	completeCtx, completeCancel := context.WithTimeout(ctx, 1*time.Minute)
	defer completeCancel()

	completeResult, err := client.CompleteMultipartUpload(completeCtx, &oss.CompleteMultipartUploadRequest{
		Bucket:   oss.Ptr(bucketName),
		Key:      oss.Ptr(objectName),
		UploadId: oss.Ptr(uploadId),
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return "", fmt.Errorf("完成分片上传失败: %w", err)
	}

	uploadSuccess = true
	logx.Infof("分片上传完成: Bucket=%s, Key=%s, ETag=%s",
		*completeResult.Bucket, *completeResult.Key, *completeResult.ETag)

	return objectName, nil
}

// uploadPartsConfig 分片上传配置。
type uploadPartsConfig struct {
	Bucket     string
	Key        string
	UploadId   string
	FileSize   int64
	PartSize   int64
	TotalParts int64
}

// uploadPartsParallel 并发上传分片。
func uploadPartsParallel(ctx context.Context, client *oss.Client, file fileReader, config *uploadPartsConfig) ([]oss.UploadPart, error) {
	// 创建结果切片（预分配空间）
	parts := make([]oss.UploadPart, config.TotalParts)

	// 创建错误通道
	errChan := make(chan error, config.TotalParts)

	// 使用 WaitGroup 等待所有分片完成
	var wg sync.WaitGroup

	// 创建信号量控制并发数
	sem := make(chan struct{}, common.MaxConcurrentParts)

	startTime := time.Now()

	// 启动所有分片上传任务
	for i := int64(0); i < config.TotalParts; i++ {
		wg.Add(1)

		go func(partNumber int64) {
			defer wg.Done()

			// 获取信号量（控制并发）
			sem <- struct{}{}
			defer func() { <-sem }()

			// 计算当前分片的偏移量和大小
			offset := partNumber * config.PartSize
			currentPartSize := config.PartSize
			if offset+currentPartSize > config.FileSize {
				currentPartSize = config.FileSize - offset
			}

			// 读取分片数据
			partData := make([]byte, currentPartSize)
			n, err := file.ReadAt(partData, offset)
			if err != nil && err != io.EOF {
				errChan <- fmt.Errorf("读取分片 %d 失败: %w", partNumber+1, err)
				return
			}

			// 上传分片（重试机制）
			partCtx, partCancel := context.WithTimeout(ctx, 3*time.Minute)
			defer partCancel()

			var partResult *oss.UploadPartResult
			maxRetries := 3

			for retry := 0; retry < maxRetries; retry++ {
				partStartTime := time.Now()

				partResult, err = client.UploadPart(partCtx, &oss.UploadPartRequest{
					Bucket:     oss.Ptr(config.Bucket),
					Key:        oss.Ptr(config.Key),
					UploadId:   oss.Ptr(config.UploadId),
					PartNumber: int32(partNumber + 1),
					Body:       bytes.NewReader(partData[:n]),
				})

				if err == nil {
					// 上传成功
					partDuration := time.Since(partStartTime)
					speed := float64(n) / partDuration.Seconds() / (1024 * 1024) // MB/s

					logx.Infof("分片 %d/%d 上传成功: %.2f MB, 耗时 %v, 速度 %.2f MB/s",
						partNumber+1,
						config.TotalParts,
						float64(n)/(1024*1024),
						partDuration.Round(time.Millisecond),
						speed)
					break
				}

				// 上传失败，重试
				if retry < maxRetries-1 {
					waitTime := time.Duration(retry+1) * time.Second
					logx.Errorf("分片 %d 上传失败，%v 后重试 (%d/%d): %v",
						partNumber+1, waitTime, retry+1, maxRetries, err)
					time.Sleep(waitTime)
				} else {
					errChan <- fmt.Errorf("分片 %d 上传失败（已重试 %d 次）: %w",
						partNumber+1, maxRetries, err)
					return
				}
			}

			// 记录已上传的分片信息
			parts[partNumber] = oss.UploadPart{
				PartNumber: int32(partNumber + 1),
				ETag:       partResult.ETag,
			}

		}(i)
	}

	// 等待所有分片上传完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	if err := <-errChan; err != nil {
		return nil, err
	}

	totalDuration := time.Since(startTime)
	avgSpeed := float64(config.FileSize) / totalDuration.Seconds() / (1024 * 1024)
	logx.Infof("所有分片上传完成: 总耗时 %v, 平均速度 %.2f MB/s",
		totalDuration.Round(time.Millisecond), avgSpeed)

	return parts, nil
}

// fileReader 文件读取接口（支持 ReadAt）。
type fileReader interface {
	io.ReaderAt
	io.Closer
}

// openFile 打开文件的辅助函数。
func openFile(filePath string) (fileReader, error) {
	return os.Open(filePath)
}
