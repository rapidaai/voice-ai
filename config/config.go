package config

type RegionalHost struct {
	Region string `mapstructure:"region" validate:"required"`
	Host   string `mapstructure:"host" validate:"required"`
}

type ServiceHostConfig struct {
	Host    string         `mapstructure:"host" validate:"required"`
	Public  string         `mapstructure:"public"`
	Regions []RegionalHost `mapstructure:"regions"`
}

type AppConfig struct {
	//
	Name      string `mapstructure:"service_name" validate:"required"`
	ServiceID uint64 `mapstructure:"service_id" validate:"required"`
	Version   string `mapstructure:"version"`
	Host      string `mapstructure:"host" validate:"required"`
	Env       string `mapstructure:"env" validate:"required"`
	Port      int    `mapstructure:"port" validate:"required"`
	LogLevel  string `mapstructure:"log_level" validate:"required"`
	Secret    string `mapstructure:"secret" validate:"required"`

	Integration ServiceHostConfig `mapstructure:"integration"`
	Endpoint    ServiceHostConfig `mapstructure:"endpoint"`
	Assistant   ServiceHostConfig `mapstructure:"assistant"`
	Web         ServiceHostConfig `mapstructure:"web"`
	Document    ServiceHostConfig `mapstructure:"document"`
	Ui          ServiceHostConfig `mapstructure:"ui"`
}

func (cfg *AppConfig) IsDevelopment() bool {
	return cfg.Env != "production"
}

func (cfg *AppConfig) BaseUrl() string {
	return cfg.Ui.Host
}
