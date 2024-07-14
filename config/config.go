package config

import (
	"log"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/lexatic/web-backend/pkg/configs"
	"github.com/spf13/viper"
)

// Application config structure
type AppConfig struct {
	Name           string                 `mapstructure:"service_name" validate:"required"`
	Version        string                 `mapstructure:"version" validate:"required"`
	Host           string                 `mapstructure:"host" validate:"required"`
	Env            string                 `mapstructure:"env" validate:"required"`
	Secret         string                 `mapstructure:"secret" validate:"required"`
	Port           int                    `mapstructure:"port" validate:"required"`
	LogLevel       string                 `mapstructure:"log_level" validate:"required"`
	PostgresConfig configs.PostgresConfig `mapstructure:"postgres" validate:"required"`
	RedisConfig    configs.RedisConfig    `mapstructure:"redis" validate:"required"`

	// all the host
	ProviderHost    string `mapstructure:"provider_host" validate:"required"`
	IntegrationHost string `mapstructure:"integration_host" validate:"required"`
	EndpointHost    string `mapstructure:"endpoint_host" validate:"required"`
	WorkflowHost    string `mapstructure:"workflow_host" validate:"required"`
	WebhookHost     string `mapstructure:"webhook_host" validate:"required"`
	WebHost         string `mapstructure:"web_host" validate:"required"`
	ExperimentHost  string `mapstructure:"experiment_host" validate:"required"`

	AssetStoreConfig AssetStoreConfig `mapstructure:"asset_store" validate:"required"`

	GoogleClientId     string `mapstructure:"google_client_id" validate:"required"`
	GoogleClientSecret string `mapstructure:"google_client_secret" validate:"required"`

	LinkedinClientId     string `mapstructure:"linkedin_client_id" validate:"required"`
	LinkedinClientSecret string `mapstructure:"linkedin_client_secret" validate:"required"`

	GithubClientId     string `mapstructure:"github_client_id" validate:"required"`
	GithubClientSecret string `mapstructure:"github_client_secret" validate:"required"`

	NotionClientId     string `mapstructure:"notion_client_id" validate:"required"`
	NotionClientSecret string `mapstructure:"notion_client_id" validate:"required"`

	MicrosoftClientId     string `mapstructure:"microsoft_client_id" validate:"required"`
	MicrosoftClientSecret string `mapstructure:"microsoft_client_id" validate:"required"`

	AtlassianClientId     string `mapstructure:"atlassian_client_id" validate:"required"`
	AtlassianClientSecret string `mapstructure:"atlassian_client_id" validate:"required"`

	GitlabClientId     string `mapstructure:"gitlab_client_id" validate:"required"`
	GitlabClientSecret string `mapstructure:"gitlab_client_id" validate:"required"`
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

// reading config and intializing configs for application
func InitConfig() (*viper.Viper, error) {
	vConfig := viper.NewWithOptions(viper.KeyDelimiter("__"))

	vConfig.AddConfigPath(".")
	vConfig.SetConfigName(".env")
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
	// setting all default values
	// keeping watch on https://github.com/spf13/viper/issues/188

	v.SetDefault("SERVICE_NAME", "go-service-template")
	v.SetDefault("VERSION", "0.0.1")
	v.SetDefault("HOST", "0.0.0.0")
	v.SetDefault("PORT", 9090)
	v.SetDefault("LOG_LEVEL", "debug")
	v.SetDefault("PROVIDER_HOST", "")
	v.SetDefault("INTEGRATION_HOST", "")
	//

	v.SetDefault("POSTGRES__HOST", "localhost")
	v.SetDefault("POSTGRES__PORT", 5432)
	v.SetDefault("POSTGRES__DB_NAME", "<>")
	v.SetDefault("POSTGRES__AUTH__USER", "<>")
	v.SetDefault("POSTGRES__AUTH__PASSWORD", "<>")
	v.SetDefault("POSTGRES__MAX_OPEN_CONNECTION", 10)
	v.SetDefault("POSTGRES__MAX_IDEAL_CONNECTION", 10)
	v.SetDefault("POSTGRES__SSL_MODE", "disable")

	// oauth credentials
	v.SetDefault("GITHUB_CLIENT_ID", "1b39b8c6caf9337f7473")
	v.SetDefault("GITHUB_CLIENT_SECRET", "7bd6ed9412ae57406e19ebd1248384ba7fb86e0d")
	v.SetDefault("GOOGLE_CLIENT_ID", "269014990564-e42nboqc22jem4p3pta1s5oveml975m8.apps.googleusercontent.com")
	v.SetDefault("GOOGLE_CLIENT_SECRET", "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")
	v.SetDefault("LINKEDIN_CLIENT_ID", "86ytrlc3k8p3f4")
	v.SetDefault("LINKEDIN_CLIENT_SECRET", "Jvlt9jnMXY1ov4Xr")

}

// Getting application config from viper
func GetApplicationConfig(v *viper.Viper) (*AppConfig, error) {
	var config AppConfig
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
