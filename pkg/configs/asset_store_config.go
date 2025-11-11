package configs

type StorageType string

const (
	S3    StorageType = "s3"
	LOCAL StorageType = "local"
	CDN   StorageType = "cdn"
)

// asset_upload_bucket

type AssetStoreConfig struct {
	StorageType       string     `mapstructure:"storage_type" validate:"required"`
	StoragePathPrefix string     `mapstructure:"storage_path_prefix"`
	Auth              *AwsConfig `mapstructure:"auth"`
}

func (cfg *AssetStoreConfig) Type() StorageType {
	if cfg.StorageType == string(S3) {
		return S3
	}
	return LOCAL
}

func (cfg *AssetStoreConfig) IsLocal() bool {
	return cfg.Type() != S3
}

func (cfg *AssetStoreConfig) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"storage_type":        cfg.StorageType,
		"storage_path_prefix": cfg.StoragePathPrefix,
	}

	if cfg.Auth != nil {
		result["auth"] = cfg.Auth.ToMap()
	} else {
		result["auth"] = nil
	}

	return result
}
