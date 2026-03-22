package utils

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// OSSStorage OSS 存储实现
type OSSStorage struct {
	client *oss.Client
	bucket string
}

// newOSSStorage 创建 OSS 存储实例
func newOSSStorage(cfg StorageConfig) (*OSSStorage, error) {
	if err := ossLoadEnv(); err != nil {
		return nil, fmt.Errorf("failed to load OSS env: %w", err)
	}

	provider := credentials.NewEnvironmentVariableCredentialsProvider()
	oscfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(provider).
		WithRegion(OSSRegionValue())

	client := oss.NewClient(oscfg)

	return &OSSStorage{
		client: client,
		bucket: OSSBucketNameValue(),
	}, nil
}

// PutObject 上传单个对象
func (s *OSSStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket:      oss.Ptr(s.bucket),
		Key:         oss.Ptr(key),
		Body:        body,
		ContentType: oss.Ptr(contentType),
	})
	return err
}

// GetObject 下载对象
func (s *OSSStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(s.bucket),
		Key:    oss.Ptr(key),
	})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

// DeleteObject 删除单个对象
func (s *OSSStorage) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(s.bucket),
		Key:    oss.Ptr(key),
	})
	return err
}

// InitiateMultipartUpload 初始化分片上传
func (s *OSSStorage) InitiateMultipartUpload(ctx context.Context, key string, contentType string) (string, error) {
	result, err := s.client.InitiateMultipartUpload(ctx, &oss.InitiateMultipartUploadRequest{
		Bucket:      oss.Ptr(s.bucket),
		Key:         oss.Ptr(key),
		ContentType: oss.Ptr(contentType),
	})
	if err != nil {
		return "", err
	}
	return *result.UploadId, nil
}

// UploadPart 上传单个分片
func (s *OSSStorage) UploadPart(ctx context.Context, key string, uploadID string, partNumber int, body io.Reader, size int64) (string, error) {
	result, err := s.client.UploadPart(ctx, &oss.UploadPartRequest{
		Bucket:        oss.Ptr(s.bucket),
		Key:           oss.Ptr(key),
		UploadId:      oss.Ptr(uploadID),
		PartNumber:    int32(partNumber),
		ContentLength: oss.Ptr(size),
		Body:          body,
	})
	if err != nil {
		return "", err
	}
	return oss.ToString(result.ETag), nil
}

// CompleteMultipartUpload 完成分片上传
func (s *OSSStorage) CompleteMultipartUpload(ctx context.Context, key string, uploadID string, parts []Part) error {
	// 转换 Part 结构
	ossParts := make([]oss.UploadPart, len(parts))
	for i, p := range parts {
		ossParts[i] = oss.UploadPart{
			PartNumber: int32(p.PartNumber),
			ETag:       oss.Ptr(p.ETag),
		}
	}

	_, err := s.client.CompleteMultipartUpload(ctx, &oss.CompleteMultipartUploadRequest{
		Bucket:   oss.Ptr(s.bucket),
		Key:      oss.Ptr(key),
		UploadId: oss.Ptr(uploadID),
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
			Parts: ossParts,
		},
	})
	return err
}

// AbortMultipartUpload 取消分片上传
func (s *OSSStorage) AbortMultipartUpload(ctx context.Context, key string, uploadID string) error {
	_, err := s.client.AbortMultipartUpload(ctx, &oss.AbortMultipartUploadRequest{
		Bucket:   oss.Ptr(s.bucket),
		Key:      oss.Ptr(key),
		UploadId: oss.Ptr(uploadID),
	})
	return err
}

// GetPresignedURL 生成预签名 URL
func (s *OSSStorage) GetPresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	result, err := s.client.Presign(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(s.bucket),
		Key:    oss.Ptr(key),
	}, oss.PresignExpires(expires))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

// Close 关闭存储连接
func (s *OSSStorage) Close() error {
	return nil
}
