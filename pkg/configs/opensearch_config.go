package configs

type OpenSearchConfig struct {
	Schema        string    `mapstructure:"schema" validate:"required"`
	Host          string    `mapstructure:"host" validate:"required"`
	Port          int       `mapstructure:"port"`
	Auth          BasicAuth `mapstructure:"auth"`
	MaxRetries    int       `mapstructure:"max_retries" validate:"required"`
	MaxConnection int       `mapstructure:"max_connection" validate:"required"`
}
