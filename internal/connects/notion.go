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
	NOTION_WORKPLACE_SCOPE   = []string{}
	NOTION_WORKPLACE_CONNECT = "/connect-knowledge/notion"
)

func NewNotionWorkplaceConnect(cfg *config.AppConfig, logger commons.Logger) NotionConnect {
	return NotionConnect{
		notionOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), NOTION_WORKPLACE_CONNECT),
			ClientID:     cfg.NotionClientId,
			ClientSecret: cfg.NotionClientSecret,
			Scopes:       NOTION_WORKPLACE_SCOPE,
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://api.notion.com/v1/oauth/authorize",
				TokenURL:  "https://api.notion.com/v1/oauth/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		logger: logger,
	}
}

func (notionConnect *NotionConnect) AuthCodeURL(state string) string {
	return notionConnect.notionOauthConfig.AuthCodeURL(state)
}
