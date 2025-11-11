package storage_files

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/storages"
)

type localFileStorage struct {
	config configs.AssetStoreConfig
	logger commons.Logger
}

func (lfs *localFileStorage) Name() string {
	return "local"
}

// Get implements storages.Storage.
func (lfs *localFileStorage) Get(ctx context.Context, key string) storages.GetStorageOutput {
	lfs.logger.Debugf("localstorage.get with file path name %s", key)

	filePath := path.Join(lfs.config.StoragePathPrefix, key)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		lfs.logger.Errorf("File does not exist: %s", filePath)
		return storages.GetStorageOutput{Error: err}
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return storages.GetStorageOutput{Error: err}
	}
	return storages.GetStorageOutput{Data: content}
}

func NewLocalFileStorage(cfg configs.AssetStoreConfig, logger commons.Logger) storages.Storage {
	return &localFileStorage{
		config: cfg,
		logger: logger,
	}
}

// Store implements storages.Storage.
func (lfs *localFileStorage) Store(ctx context.Context, key string, fileContent []byte) storages.StorageOutput {
	lfs.logger.Debugf("localstorage.store with file path name %s", key)
	completePath := path.Join(lfs.config.StoragePathPrefix, key)
	err := os.MkdirAll(filepath.Dir(path.Join(lfs.config.StoragePathPrefix, key)), 0755)
	if err != nil {
		lfs.logger.Errorf("unable to create complete path, err %v", err)
		return storages.StorageOutput{
			CompletePath: completePath,
			StorageType:  configs.LOCAL,
			Error:        err,
		}
	}

	err = os.WriteFile(path.Join(lfs.config.StoragePathPrefix, key), fileContent, 0644)
	if err != nil {
		lfs.logger.Errorf("unable to store a file to local path, err %v", err)
		return storages.StorageOutput{
			CompletePath: completePath,
			StorageType:  configs.LOCAL,
			Error:        err,
		}
	}
	return storages.StorageOutput{
		CompletePath: completePath,
		StorageType:  configs.LOCAL,
	}

}

func (lfs *localFileStorage) GetUrl(ctx context.Context, key string) storages.StorageOutput {
	lfs.logger.Debugf("localstorage.getUrl with file path name %s", key)
	return storages.StorageOutput{
		CompletePath: path.Join("file://", lfs.config.StoragePathPrefix, key),
		StorageType:  configs.LOCAL,
	}
}
