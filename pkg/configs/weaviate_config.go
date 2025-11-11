package configs

type WeaviateConfig struct {
	Host   string     `mapstructure:"host"`
	Scheme string     `mapstructure:"scheme"`
	Auth   ApiKeyAuth `mapstructure:"auth"`
}
