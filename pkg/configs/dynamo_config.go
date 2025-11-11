package configs

type DynamoConfig struct {
	Auth       AwsConfig `mapstructure:"auth"`
	MaxRetries int       `mapstructure:"max_retries" validate:"required"`
}
