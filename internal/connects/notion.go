package internal_connects

import (
	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/linkedin"
)

type NotionConnect struct {
	logger              commons.Logger
	linkedinOauthConfig oauth2.Config
}

func NewNotionConnect(cfg *config.AppConfig, logger commons.Logger) NotionConnect {
	return NotionConnect{
		linkedinOauthConfig: oauth2.Config{
			RedirectURL:  "https://www.rapida.ai/auth/signin",
			ClientID:     cfg.NotionClientId,
			ClientSecret: cfg.NotionClientSecret,
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     linkedin.Endpoint,
		},
		logger: logger,
	}
}
