package utils

import (
	"sync"
)

// StorageManager 存储管理器（单例）
type StorageManager struct {
	mu      sync.RWMutex
	storage ObjectStorage
}

var storageManager = &StorageManager{}

// InitStorage 初始化存储
func InitStorage(cfg StorageConfig) error {
	storage, err := NewObjectStorage(cfg)
	if err != nil {
		return err
	}

	storageManager.mu.Lock()
	defer storageManager.mu.Unlock()

	if storageManager.storage != nil {
		_ = storageManager.storage.Close()
	}

	storageManager.storage = storage
	return nil
}

// GetStorage 获取存储实例
func GetStorage() (ObjectStorage, error) {
	storageManager.mu.RLock()
	defer storageManager.mu.RUnlock()

	if storageManager.storage == nil {
		return nil, ErrStorageNotInitialized
	}

	return storageManager.storage, nil
}

// CloseStorage 关闭存储
func CloseStorage() error {
	storageManager.mu.Lock()
	defer storageManager.mu.Unlock()

	if storageManager.storage != nil {
		err := storageManager.storage.Close()
		storageManager.storage = nil
		return err
	}

	return nil
}
