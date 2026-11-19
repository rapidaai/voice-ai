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

type OAuth2Config struct {
	GoogleClientId     string `mapstructure:"google_client_id"`
	GoogleClientSecret string `mapstructure:"google_client_secret"`

	LinkedinClientId     string `mapstructure:"linkedin_client_id"`
	LinkedinClientSecret string `mapstructure:"linkedin_client_secret"`

	GithubClientId     string `mapstructure:"github_client_id"`
	GithubClientSecret string `mapstructure:"github_client_secret"`

	MicrosoftClientId     string `mapstructure:"microsoft_client_id"`
	MicrosoftClientSecret string `mapstructure:"microsoft_client_secret"`

	AtlassianClientId     string `mapstructure:"atlassian_client_id"`
	AtlassianClientSecret string `mapstructure:"atlassian_client_secret"`

	GitlabClientId     string `mapstructure:"gitlab_client_id"`
	GitlabClientSecret string `mapstructure:"gitlab_client_secret"`

	NotionClientId     string `mapstructure:"notion_client_id"`
	NotionClientSecret string `mapstructure:"notion_client_secret"`

	SlackAppId              string `mapstructure:"slack_app_id"`
	SlackClientId           string `mapstructure:"slack_client_id"`
	SlackClientSecret       string `mapstructure:"slack_client_secret"`
	SlackSigningSecret      string `mapstructure:"slack_signing_secret"`
	SlackVerificationSecret string `mapstructure:"slack_verification_secret"`

	HubspotClientId     string `mapstructure:"hubspot_client_id"`
	HubspotClientSecret string `mapstructure:"hubspot_client_secret"`
}

type WebAppConfig struct {
	config.AppConfig `mapstructure:",squash"`
	PostgresConfig   configs.PostgresConfig   `mapstructure:"postgres" validate:"required"`
	RedisConfig      configs.RedisConfig      `mapstructure:"redis" validate:"required"`
	AssetStoreConfig configs.AssetStoreConfig `mapstructure:"asset_store" validate:"required"`
	OAuthConfig      OAuth2Config             `mapstructure:"oauth2" validate:"required"`
	//
	EmailerConfig *configs.EmailerConfig `mapstructure:"emailer"`
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
		vConfig.SetConfigName("web")
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
func GetApplicationConfig(v *viper.Viper) (*WebAppConfig, error) {
	if v.IsSet("service_assertion") {
		return nil, pkg_errors.ApplicationConfigurationInvalidError{}
	}
	for _, key := range v.AllKeys() {
		if key == "service_assertion" || strings.HasPrefix(key, "service_assertion.") {
			return nil, pkg_errors.ApplicationConfigurationInvalidError{}
		}
	}
	var config WebAppConfig
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
