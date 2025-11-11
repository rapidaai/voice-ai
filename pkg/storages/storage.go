package storages

import (
	"context"

	"github.com/rapidaai/pkg/configs"
)

type StorageOutput struct {
	CompletePath string              `json:"complete_path"`
	StorageType  configs.StorageType `json:"storage_type"`
	Error        error               `json:"error"`
}

type GetStorageOutput struct {
	Data  []byte
	Error error
}

type Storage interface {
	Name() string
	Store(ctx context.Context, key string, fileContent []byte) StorageOutput
	Get(ctx context.Context, key string) GetStorageOutput
	GetUrl(ctx context.Context, key string) StorageOutput
}
