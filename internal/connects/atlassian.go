package internal_connects

import (
	"fmt"

	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"golang.org/x/oauth2"
)

type AtlassianConnect struct {
	logger               commons.Logger
	atlassianOauthConfig oauth2.Config
}

var (
	CONFLUENCE_CONNECT_URL = "/connect-common/atlassian"
	CONFLUENCE_SCOPE       = [...]string{
		"search:confluence",
		"read:confluence-content.summary",
		"read:confluence-content.all"}

	JIRA_SCOPE       = [...]string{}
	JIRA_CONNECT_URL = "/connect-common/atlassian"
)

func NewConfluenceConnect(cfg *config.AppConfig, logger commons.Logger) AtlassianConnect {
	return AtlassianConnect{
		atlassianOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), CONFLUENCE_CONNECT_URL),
			ClientID:     cfg.AtlassianClientId,
			ClientSecret: cfg.AtlassianClientSecret,
			Scopes:       CONFLUENCE_SCOPE[:],
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://auth.atlassian.com/authorize",
				TokenURL:  "https://auth.atlassian.com/oauth/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		logger: logger,
	}
}

func NewJiraConnect(cfg *config.AppConfig, logger commons.Logger) AtlassianConnect {
	return AtlassianConnect{
		atlassianOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), JIRA_CONNECT_URL),
			ClientID:     cfg.AtlassianClientId,
			ClientSecret: cfg.AtlassianClientSecret,
			Scopes:       JIRA_SCOPE[:],
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://auth.atlassian.com/authorize",
				TokenURL:  "https://auth.atlassian.com/oauth/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		logger: logger,
	}
}

// https://auth.atlassian.com/authorize?audience=api.atlassian.com&client_id=Et8qcoSIpSs1h1MMoRgU0rgbU9vftbCo&scope=write%3Aconfluence-content%20write%3Aconfluence-file%20readonly%3Acontent.attachment%3Aconfluence%20write%3Aconfluence-groups%20search%3Aconfluence%20read%3Aconfluence-content.summary%20read%3Aconfluence-content.all&redirect_uri=https%3A%2F%2Frapida.ai%2Fconnect%2Fatlassian&state=${YOUR_USER_BOUND_VALUE}&response_type=code&prompt=consent

func (atlassianConnect *AtlassianConnect) AuthCodeURL(state string) string {
	return atlassianConnect.atlassianOauthConfig.AuthCodeURL(state)
}
