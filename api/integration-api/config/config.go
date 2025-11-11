package config

import (
	"log"
	"os"

	"github.com/go-playground/validator/v10"
	config "github.com/rapidaai/config"
	"github.com/rapidaai/pkg/configs"
	"github.com/spf13/viper"
)

// Application config structure
type IntegrationConfig struct {
	config.AppConfig `mapstructure:",squash"`
	PostgresConfig   configs.PostgresConfig   `mapstructure:"postgres" validate:"required"`
	RedisConfig      configs.RedisConfig      `mapstructure:"redis" validate:"required"`
	SendgridApiKey   string                   `mapstructure:"sendgrid_api_key" validate:"required"`
	AssetStoreConfig configs.AssetStoreConfig `mapstructure:"asset_store" validate:"required"`
}

// reading config and intializing configs for application
func InitConfig() (*viper.Viper, error) {
	vConfig := viper.NewWithOptions(viper.KeyDelimiter("__"))

	vConfig.AddConfigPath("./env/")
	vConfig.SetConfigName(".integration.env")
	path := os.Getenv("ENV_PATH")
	if path != "" {
		log.Printf("env path %v", path)
		vConfig.SetConfigFile(path)
	}
	vConfig.SetConfigType("env")
	vConfig.AutomaticEnv()
	err := vConfig.ReadInConfig()
	if err == nil {
		log.Printf("Error while reading the config")
	}

	//
	setDefault(vConfig)
	if err = vConfig.ReadInConfig(); err != nil && !os.IsNotExist(err) {
		log.Printf("Reading from env varaibles.")
	}

	return vConfig, nil
}

func setDefault(v *viper.Viper) {
	v.SetDefault("SERVICE_NAME", "go-service-template")
	v.SetDefault("VERSION", "0.0.1")
	v.SetDefault("HOST", "0.0.0.0")
	v.SetDefault("PORT", 9090)
	v.SetDefault("LOG_LEVEL", "debug")

}

// Getting application config from viper
func GetApplicationConfig(v *viper.Viper) (*IntegrationConfig, error) {
	var config IntegrationConfig
	err := v.Unmarshal(&config)
	if err != nil {
		log.Printf("%+v\n", err)
		return nil, err
	}

	// valdating the app config
	validate := validator.New()
	err = validate.Struct(&config)
	if err != nil {
		log.Printf("%+v\n", err)
		return nil, err
	}
	return &config, nil
}
