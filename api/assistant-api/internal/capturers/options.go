package internal_capturers

import (
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/utils"
)

type CapturerOptions struct {
	Options map[string]interface{}
}

func (iatf *CapturerOptions) WithCustomOptions(custom map[string]interface{}) *CapturerOptions {
	iatf.Options = custom
	return iatf
}

// configs
type CapturerConfig interface {
	GetType() string
	GetName() string
	GetOptions() utils.Option
}

func (co *CapturerOptions) S3Config(
	overriddenOpts map[string]interface{},
) (configs.AssetStoreConfig, error) {
	cfg := configs.AssetStoreConfig{
		StorageType: "s3",
		Auth:        &configs.AwsConfig{},
	}
	v1, ok := co.Options["storage_path_prefix"]
	if ok {
		cfg.StoragePathPrefix = v1.(string)
	}
	v, ok := overriddenOpts["storage_path_prefix"]
	if ok {
		cfg.StoragePathPrefix = v.(string)
	}

	region, ok := co.Options["region"]
	if ok {
		cfg.Auth.Region = region.(string)
	}

	region_2, ok := overriddenOpts["region"]
	if ok {
		cfg.Auth.Region = region_2.(string)
	}

	assumeRole, ok := co.Options["assume_role"]
	if ok {
		cfg.Auth.AssumeRole = assumeRole.(string)
	}

	assumeRole_2, ok := overriddenOpts["assume_role"]
	if ok {
		cfg.Auth.AssumeRole = assumeRole_2.(string)
	}

	accessKeyId, ok := co.Options["access_key_id"]
	if ok {
		cfg.Auth.AccessKeyId = accessKeyId.(string)
	}

	accessKeyId_2, ok := overriddenOpts["access_key_id"]
	if ok {
		cfg.Auth.AccessKeyId = accessKeyId_2.(string)
	}

	secretKey, ok := co.Options["secret_key"]
	if ok {
		cfg.Auth.SecretKey = secretKey.(string)
	}

	secretKey_2, ok := overriddenOpts["secret_key"]
	if ok {
		cfg.Auth.SecretKey = secretKey_2.(string)
	}

	return cfg, nil
}
