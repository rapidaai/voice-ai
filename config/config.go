package config

type AppConfig struct {
	Name     string `mapstructure:"service_name" validate:"required"`
	Version  string `mapstructure:"version"`
	Host     string `mapstructure:"host" validate:"required"`
	Env      string `mapstructure:"env" validate:"required"`
	Port     int    `mapstructure:"port" validate:"required"`
	LogLevel string `mapstructure:"log_level" validate:"required"`
	Secret   string `mapstructure:"secret" validate:"required"`

	// all the host
	IntegrationHost string `mapstructure:"integration_host" validate:"required"`
	EndpointHost    string `mapstructure:"endpoint_host" validate:"required"`
	AssistantHost   string `mapstructure:"assistant_host" validate:"required"`
	WebHost         string `mapstructure:"web_host" validate:"required"`
	DocumentHost    string `mapstructure:"document_host" validate:"required"`
}

func (cfg *AppConfig) IsDevelopment() bool {
	return cfg.Env != "production"
}

func (cfg *AppConfig) BaseUrl() (baseUrl string) {
	baseUrl = "https://www.rapida.ai"
	if cfg.IsDevelopment() {
		baseUrl = "http://localhost:3000"
	}
	return
}
