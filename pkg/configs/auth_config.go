package configs

type BasicAuth struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type ApiKeyAuth struct {
	ApiKey string `mapstructure:"api_key"`
}
