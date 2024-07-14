package internal_connects

import (
	"fmt"

	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/gitlab"
)

type GitlabConnect struct {
	logger            commons.Logger
	gitlabOauthConfig oauth2.Config
}

var (
	GITLAB_AUTHENTICATION_SCOPE = []string{"user"}
	GITLAB_AUTHENTICATION_URL   = "/auth/signin"

	GITLAB_CODE_SCOPE   = []string{"repo"}
	GITLAB_CODE_CONNECT = "/connect-knowledge/gitlab"

	GITLAB_ACTION_SCOPE   = []string{}
	GITLAB_ACTION_CONNECT = "/action/gitlab"
)

func NewGitlabAuthenticationConnect(cfg *config.AppConfig, logger commons.Logger) GitlabConnect {
	return GitlabConnect{
		gitlabOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITLAB_AUTHENTICATION_URL),
			ClientID:     cfg.GitlabClientId,
			ClientSecret: cfg.GitlabClientSecret,
			Scopes:       GITLAB_AUTHENTICATION_SCOPE,
			Endpoint:     gitlab.Endpoint,
		},
		logger: logger,
	}
}

func NewGitlabCodeConnect(cfg *config.AppConfig, logger commons.Logger) GitlabConnect {
	return GitlabConnect{
		gitlabOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITLAB_CODE_CONNECT),
			ClientID:     cfg.GitlabClientId,
			ClientSecret: cfg.GitlabClientSecret,
			Scopes:       GITLAB_CODE_SCOPE,
			Endpoint:     gitlab.Endpoint,
		},
		logger: logger,
	}
}
func NewGitlabActionConnect(cfg *config.AppConfig, logger commons.Logger) GitlabConnect {
	return GitlabConnect{
		gitlabOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITLAB_ACTION_CONNECT),
			ClientID:     cfg.GitlabClientId,
			ClientSecret: cfg.GitlabClientSecret,
			Scopes:       GITLAB_ACTION_SCOPE,
			Endpoint:     gitlab.Endpoint,
		},
		logger: logger,
	}
}

func (gitlabConnect *GitlabConnect) AuthCodeURL(state string) string {
	return gitlabConnect.gitlabOauthConfig.AuthCodeURL(state)
}
