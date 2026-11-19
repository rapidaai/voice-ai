// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package config

import (
	"log"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/configs"
	pkg_errors "github.com/rapidaai/pkg/errors"
	pkg_validator "github.com/rapidaai/pkg/validator"
)

// Application config structure
type EndpointConfig struct {
	config.AppConfig `mapstructure:",squash"`
	PostgresConfig   configs.PostgresConfig   `mapstructure:"postgres" validate:"required"`
	RedisConfig      configs.RedisConfig      `mapstructure:"redis" validate:"required"`
	AssetStoreConfig configs.AssetStoreConfig `mapstructure:"asset_store" validate:"required"`
}

// reading config and intializing configs for application
func InitConfig() (*viper.Viper, error) {
	vConfig := viper.New()

	path := os.Getenv("ENV_PATH")
	if path != "" {
		log.Printf("config path %v", path)
		vConfig.SetConfigFile(path)
	} else {
		vConfig.AddConfigPath("./env/")
		vConfig.SetConfigName("endpoint")
		vConfig.SetConfigType("yaml")
	}

	if err := vConfig.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("application configuration could not be read")
		}
		return nil, pkg_errors.ApplicationConfigurationInvalidError{}
	}

	return vConfig, nil
}

// Getting application config from viper
func GetApplicationConfig(v *viper.Viper) (*EndpointConfig, error) {
	if v.IsSet("service_assertion") {
		return nil, pkg_errors.ApplicationConfigurationInvalidError{}
	}
	for _, key := range v.AllKeys() {
		if key == "service_assertion" || strings.HasPrefix(key, "service_assertion.") {
			return nil, pkg_errors.ApplicationConfigurationInvalidError{}
		}
	}
	var config EndpointConfig
	err := v.Unmarshal(&config)
	if err != nil {
		log.Printf("application configuration could not be decoded")
		return nil, pkg_errors.ApplicationConfigurationInvalidError{}
	}

	// valdating the app config
	validate := validator.New()
	err = validate.Struct(&config)
	if err != nil {
		log.Printf("application configuration validation failed")
		return nil, pkg_errors.ApplicationConfigurationInvalidError{}
	}
	if !pkg_validator.AuthenticationSecret(config.Secret) {
		return nil, pkg_errors.ApplicationConfigurationInvalidError{}
	}
	return &config, nil
}
