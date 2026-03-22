package config

import (
	"cloud_disk/core/utils"
	"fmt"
	"os"
)

// InitializeStorage 初始化存储系统
// 优先使用环境变量指定的存储类型，默认使用 OSS
func InitializeStorage() error {
	storageType := os.Getenv("STORAGE_TYPE")
	if storageType == "" {
		storageType = "tos" // 默认使用 TOS
	}

	cfg := utils.StorageConfig{
		Type: storageType,
	}

	// 根据存储类型读取不同的配置
	switch storageType {
	case "oss":
		// OSS 会从环境变量自动读取配置
		cfg.Region = utils.OSSRegionValue()
		cfg.BucketName = utils.OSSBucketNameValue()

	case "tos":
		// TOS 需要从环境变量读取配置
		cfg.Region = os.Getenv("VOLCENGINE_REGION")
		if cfg.Region == "" {
			cfg.Region = "cn-beijing"
		}

		cfg.Endpoint = os.Getenv("VOLCENGINE_ENDPOINT")
		if cfg.Endpoint == "" {
			return fmt.Errorf("VOLCENGINE_ENDPOINT not set for TOS storage")
		}

		cfg.BucketName = os.Getenv("VOLCENGINE_BUCKET_NAME")
		if cfg.BucketName == "" {
			return fmt.Errorf("VOLCENGINE_BUCKET_NAME not set for TOS storage")
		}

		cfg.AccessKeyID = os.Getenv("VOLCENGINE_ACCESS_KEY_ID")
		if cfg.AccessKeyID == "" {
			return fmt.Errorf("VOLCENGINE_ACCESS_KEY_ID not set for TOS storage")
		}

		cfg.AccessKeySecret = os.Getenv("VOLCENGINE_SECRET_ACCESS_KEY")
		if cfg.AccessKeySecret == "" {
			return fmt.Errorf("VOLCENGINE_SECRET_ACCESS_KEY not set for TOS storage")
		}

	default:
		return fmt.Errorf("unsupported storage type: %s", storageType)
	}

	if err := utils.InitStorage(cfg); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	fmt.Printf("Storage initialized: type=%s, region=%s, bucket=%s\n", storageType, cfg.Region, cfg.BucketName)
	return nil
}
