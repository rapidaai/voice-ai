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

type OAuthConfig struct {
	GoogleClientId     string `mapstructure:"google_client_id" validate:"required"`
	GoogleClientSecret string `mapstructure:"google_client_secret" validate:"required"`

	LinkedinClientId     string `mapstructure:"linkedin_client_id" validate:"required"`
	LinkedinClientSecret string `mapstructure:"linkedin_client_secret" validate:"required"`

	GithubClientId     string `mapstructure:"github_client_id" validate:"required"`
	GithubClientSecret string `mapstructure:"github_client_secret" validate:"required"`

	NotionClientId     string `mapstructure:"notion_client_id" validate:"required"`
	NotionClientSecret string `mapstructure:"notion_client_secret" validate:"required"`

	MicrosoftClientId     string `mapstructure:"microsoft_client_id" validate:"required"`
	MicrosoftClientSecret string `mapstructure:"microsoft_client_secret" validate:"required"`

	AtlassianClientId     string `mapstructure:"atlassian_client_id" validate:"required"`
	AtlassianClientSecret string `mapstructure:"atlassian_client_secret" validate:"required"`

	GitlabClientId     string `mapstructure:"gitlab_client_id" validate:"required"`
	GitlabClientSecret string `mapstructure:"gitlab_client_secret" validate:"required"`

	SlackAppId              string `mapstructure:"slack_app_id" validate:"required"`
	SlackClientId           string `mapstructure:"slack_client_id" validate:"required"`
	SlackClientSecret       string `mapstructure:"slack_client_secret" validate:"required"`
	SlackSigningSecret      string `mapstructure:"slack_signing_secret" validate:"required"`
	SlackVerificationSecret string `mapstructure:"slack_verification_secret" validate:"required"`

	HubspotClientId     string `mapstructure:"hubspot_client_id" validate:"required"`
	HubspotClientSecret string `mapstructure:"hubspot_client_secret" validate:"required"`
}

type WebAppConfig struct {
	config.AppConfig `mapstructure:",squash"`
	PostgresConfig   configs.PostgresConfig   `mapstructure:"postgres" validate:"required"`
	RedisConfig      configs.RedisConfig      `mapstructure:"redis" validate:"required"`
	AssetStoreConfig configs.AssetStoreConfig `mapstructure:"asset_store" validate:"required"`
	OAuthConfig      `mapstructure:",squash"`
}

// reading config and intializing configs for application
func InitConfig() (*viper.Viper, error) {
	vConfig := viper.NewWithOptions(viper.KeyDelimiter("__"))

	vConfig.AddConfigPath("./env/")
	vConfig.SetConfigName(".web.env")
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

	if err = vConfig.ReadInConfig(); err != nil && !os.IsNotExist(err) {
		log.Printf("Reading from env varaibles.")
	}

	return vConfig, nil
}

// Getting application config from viper
func GetApplicationConfig(v *viper.Viper) (*WebAppConfig, error) {
	var config WebAppConfig
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
