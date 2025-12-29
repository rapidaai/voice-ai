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
	if err := vConfig.ReadInConfig(); err != nil {
		log.Printf("Error while reading the config")
	}

	if err := vConfig.ReadInConfig(); err != nil && !os.IsNotExist(err) {
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
