package configs

type RedisConfig struct {
	Host               string    `mapstructure:"host" validate:"required"`
	Port               int       `mapstructure:"port" validate:"required"`
	Db                 int       `mapstructure:"db"`
	MaxConnection      int       `mapstructure:"max_connection" validate:"required"`
	Auth               BasicAuth `mapstructure:"auth"`
	InsecureSkipVerify bool      `mapstructure:"insecure_skip_verify"`
}
