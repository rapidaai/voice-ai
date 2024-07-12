package internal_connects

import (
	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/linkedin"
)

type GitlabConnect struct {
	logger              commons.Logger
	linkedinOauthConfig oauth2.Config
}

func NewGitlabConnect(cfg *config.AppConfig, logger commons.Logger) GitlabConnect {
	return GitlabConnect{
		linkedinOauthConfig: oauth2.Config{
			RedirectURL:  "https://www.rapida.ai/auth/signin",
			ClientID:     cfg.GitlabClientId,
			ClientSecret: cfg.GitlabClientSecret,
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     linkedin.Endpoint,
		},
		logger: logger,
	}
}
