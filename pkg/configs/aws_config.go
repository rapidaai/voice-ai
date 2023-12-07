package configs

type AwsConfig struct {
	Region     string `mapstructure:"region" validate:"required"`
	AssumeRole string `mapstructure:"assume_role"`
}
