package internal_connects

import (
	"fmt"

	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"golang.org/x/oauth2"
)

type NotionConnect struct {
	logger            commons.Logger
	notionOauthConfig oauth2.Config
}

var (

	// MICROSOFT_DRIVE_STATE       = "connect/"
	NOTION_WORKPLACE_SCOPE   = []string{"email", "profile"}
	NOTION_WORKPLACE_CONNECT = "/connect/one-drive"
)

func NewNotionWorkplaceConnect(cfg *config.AppConfig, logger commons.Logger) NotionConnect {
	return NotionConnect{
		notionOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), NOTION_WORKPLACE_CONNECT),
			ClientID:     cfg.NotionClientId,
			ClientSecret: cfg.NotionClientSecret,
			Scopes:       NOTION_WORKPLACE_SCOPE,
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://auth.atlassian.com/authorize",
				TokenURL:  "https://auth.atlassian.com/oauth/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		logger: logger,
	}
}
