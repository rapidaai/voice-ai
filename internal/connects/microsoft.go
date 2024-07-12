package internal_connects

import (
	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/linkedin"
)

type MicrosoftConnect struct {
	logger              commons.Logger
	linkedinOauthConfig oauth2.Config
}

func NewMicrosoftConnect(cfg *config.AppConfig, logger commons.Logger) MicrosoftConnect {
	return MicrosoftConnect{
		linkedinOauthConfig: oauth2.Config{
			RedirectURL:  "https://www.rapida.ai/auth/signin",
			ClientID:     cfg.MicrosoftClientId,
			ClientSecret: cfg.MicrosoftClientSecret,
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     linkedin.Endpoint,
		},
		logger: logger,
	}
}
