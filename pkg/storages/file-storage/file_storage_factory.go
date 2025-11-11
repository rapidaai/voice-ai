package storage_files

import (
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/storages"
)

func NewStorage(config configs.AssetStoreConfig, logger commons.Logger) storages.Storage {
	switch config.Type() {
	case configs.S3:
		return NewAwsFileStorage(config, logger)
	case configs.LOCAL:
		return NewLocalFileStorage(config, logger)
	case configs.CDN:
		return NewCDNStorage(config, logger)
	default:
		logger.Warnf("illegal/unsupported storage type, %s", config.StorageType)
		return NewLocalFileStorage(config, logger)
	}
}
