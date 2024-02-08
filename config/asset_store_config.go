package config

import "github.com/lexatic/web-backend/pkg/configs"

type AssetStoreConfig struct {
	AssetUploadBucket string            `mapstructure:"asset_upload_bucket" validate:"required"`
	Auth              configs.AwsConfig `mapstructure:"auth"`
}
