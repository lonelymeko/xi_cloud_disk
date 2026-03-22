package utils

import (
	"errors"
)

var (
	ErrUnsupportedStorageType = errors.New("unsupported storage type")
	ErrStorageNotInitialized  = errors.New("storage not initialized")
)
