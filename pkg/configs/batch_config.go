package configs

type BatchConfig struct {
	BatchType   string     `mapstructure:"batch_type"`
	BatchScript string     `mapstructure:"batch_script"`
	Auth        *AwsConfig `mapstructure:"auth"`
}

func (b *BatchConfig) IsLocal() bool {
	return b.BatchType != "aws"
}
