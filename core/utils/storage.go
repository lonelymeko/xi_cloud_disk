package utils

import (
	"context"
	"io"
	"time"
)

// ObjectStorage 统一的对象存储接口
type ObjectStorage interface {
	// PutObject 上传单个对象
	PutObject(ctx context.Context, key string, body io.Reader, size int64, contentType string) error

	// GetObject 下载对象
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)

	// DeleteObject 删除单个对象
	DeleteObject(ctx context.Context, key string) error

	// InitiateMultipartUpload 初始化分片上传
	InitiateMultipartUpload(ctx context.Context, key string, contentType string) (uploadID string, err error)

	// UploadPart 上传单个分片
	UploadPart(ctx context.Context, key string, uploadID string, partNumber int, body io.Reader, size int64) (etag string, err error)

	// CompleteMultipartUpload 完成分片上传
	CompleteMultipartUpload(ctx context.Context, key string, uploadID string, parts []Part) error

	// AbortMultipartUpload 取消分片上传
	AbortMultipartUpload(ctx context.Context, key string, uploadID string) error

	// GetPresignedURL 生成预签名 URL
	GetPresignedURL(ctx context.Context, key string, expires time.Duration) (string, error)

	// Close 关闭存储连接
	Close() error
}

// Part 代表分片信息
type Part struct {
	PartNumber int
	ETag       string
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type            string // "oss" 或 "tos"
	Region          string
	Endpoint        string
	PresignEndpoint string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
}

// NewObjectStorage 根据配置创建存储实例
func NewObjectStorage(cfg StorageConfig) (ObjectStorage, error) {
	switch cfg.Type {
	case "oss":
		return newOSSStorage(cfg)
	case "tos":
		return newTOSStorage(cfg)
	default:
		return nil, ErrUnsupportedStorageType
	}
}
