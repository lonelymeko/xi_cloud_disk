package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
)

// TOSStorage TOS 存储实现
type TOSStorage struct {
	client *tos.ClientV2
	bucket string
}

// newTOSStorage 创建 TOS 存储实例
func newTOSStorage(cfg StorageConfig) (*TOSStorage, error) {
	if cfg.Endpoint == "" || cfg.Region == "" || cfg.BucketName == "" {
		return nil, fmt.Errorf("TOS storage requires Endpoint, Region, and BucketName")
	}

	// 创建 TOS 客户端
	client, err := tos.NewClientV2(cfg.Endpoint,
		tos.WithRegion(cfg.Region),
		tos.WithCredentials(tos.NewStaticCredentials(cfg.AccessKeyID, cfg.AccessKeySecret)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TOS client: %w", err)
	}

	return &TOSStorage{
		client: client,
		bucket: cfg.BucketName,
	}, nil
}

// PutObject 上传单个对象
func (s *TOSStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObjectV2(ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket:      s.bucket,
			Key:         key,
			ContentType: contentType,
		},
		Content: body,
	})
	return err
}

// GetObject 下载对象
func (s *TOSStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := s.client.GetObjectV2(ctx, &tos.GetObjectV2Input{
		Bucket: s.bucket,
		Key:    key,
	})
	if err != nil {
		return nil, err
	}
	return result.Content, nil
}

// DeleteObject 删除单个对象
func (s *TOSStorage) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObjectV2(ctx, &tos.DeleteObjectV2Input{
		Bucket: s.bucket,
		Key:    key,
	})
	return err
}

// InitiateMultipartUpload 初始化分片上传
func (s *TOSStorage) InitiateMultipartUpload(ctx context.Context, key string, contentType string) (string, error) {
	result, err := s.client.CreateMultipartUploadV2(ctx, &tos.CreateMultipartUploadV2Input{
		Bucket:      s.bucket,
		Key:         key,
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return result.UploadID, nil
}

// UploadPart 上传单个分片
func (s *TOSStorage) UploadPart(ctx context.Context, key string, uploadID string, partNumber int, body io.Reader, size int64) (string, error) {
	// 读取 body 内容
	data, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("failed to read part data: %w", err)
	}

	result, err := s.client.UploadPartV2(ctx, &tos.UploadPartV2Input{
		UploadPartBasicInput: tos.UploadPartBasicInput{
			Bucket:     s.bucket,
			Key:        key,
			UploadID:   uploadID,
			PartNumber: partNumber,
		},
		Content:       io.NopCloser(bytes.NewReader(data)),
		ContentLength: int64(len(data)),
	})
	if err != nil {
		return "", err
	}
	return result.ETag, nil
}

// CompleteMultipartUpload 完成分片上传
func (s *TOSStorage) CompleteMultipartUpload(ctx context.Context, key string, uploadID string, parts []Part) error {
	// 转换 Part 结构
	tosParts := make([]tos.UploadedPartV2, len(parts))
	for i, p := range parts {
		tosParts[i] = tos.UploadedPartV2{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		}
	}

	_, err := s.client.CompleteMultipartUploadV2(ctx, &tos.CompleteMultipartUploadV2Input{
		Bucket:   s.bucket,
		Key:      key,
		UploadID: uploadID,
		Parts:    tosParts,
	})
	return err
}

// AbortMultipartUpload 取消分片上传
func (s *TOSStorage) AbortMultipartUpload(ctx context.Context, key string, uploadID string) error {
	_, err := s.client.AbortMultipartUpload(ctx, &tos.AbortMultipartUploadInput{
		Bucket:   s.bucket,
		Key:      key,
		UploadID: uploadID,
	})
	return err
}

// GetPresignedURL 生成预签名 URL
func (s *TOSStorage) GetPresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	result, err := s.client.PreSignedURL(&tos.PreSignedURLInput{
		HTTPMethod: http.MethodGet,
		Bucket:     s.bucket,
		Key:        key,
		Expires:    int64(expires.Seconds()),
	})
	if err != nil {
		return "", err
	}
	return result.SignedUrl, nil
}

// Close 关闭存储连接
func (s *TOSStorage) Close() error {
	s.client.Close()
	return nil
}
