package config

import (
	"log"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/configs"
	"github.com/spf13/viper"
)

// Application config structure
type AssistantConfig struct {
	config.AppConfig `mapstructure:",squash"`
	PostgresConfig   configs.PostgresConfig   `mapstructure:"postgres" validate:"required"`
	RedisConfig      configs.RedisConfig      `mapstructure:"redis" validate:"required"`
	OpenSearchConfig configs.OpenSearchConfig `mapstructure:"opensearch" validate:"required"`
	WeaviateConfig   configs.WeaviateConfig   `mapstructure:"weaviate" validate:"required"`
	AssetStoreConfig configs.AssetStoreConfig `mapstructure:"asset_store" validate:"required"`
}

// reading config and intializing configs for application
func InitConfig() (*viper.Viper, error) {
	vConfig := viper.NewWithOptions(viper.KeyDelimiter("__"))

	vConfig.AddConfigPath("./env/")
	vConfig.SetConfigName(".assistant.env")
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
}

// Getting application config from viper
func GetApplicationConfig(v *viper.Viper) (*AssistantConfig, error) {
	var config AssistantConfig
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
