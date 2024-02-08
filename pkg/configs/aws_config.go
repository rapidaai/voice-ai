package configs

type AwsConfig struct {
	Region      string `mapstructure:"region" validate:"required"`
	AssumeRole  string `mapstructure:"assume_role"`
	AccessKeyId string `mapstructure:"access_key_id"`
	SecretKey   string `mapstructure:"secret_key"`
}
