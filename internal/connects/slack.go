package internal_connects

import (
	"context"
	"fmt"

	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"golang.org/x/oauth2"
)

type SlackConnect struct {
	ExternalConnect
	logger           commons.Logger
	slackOauthConfig oauth2.Config
}

var (

	// MICROSOFT_DRIVE_STATE       = "connect/"
	SLACK_SEND_MESSAGE_SCOPE = []string{"chat:write"}
	// Scopes:
	SLACK_SEND_MESSAGE_CONNECT = "/connect-action/slack"
)

func NewSlackActionConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) SlackConnect {
	return SlackConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		slackOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", "https://7c07-2401-4900-1cbd-4c13-3408-70d2-3232-43f2.ngrok-free.app", SLACK_SEND_MESSAGE_CONNECT),
			ClientID:     cfg.SlackClientId,
			ClientSecret: cfg.SlackClientSecret,
			Scopes:       SLACK_SEND_MESSAGE_SCOPE,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://slack.com/oauth/v2/authorize",
				TokenURL: "https://slack.com/api/oauth.v2.access",
			},
		},
		logger: logger,
	}
}

func (slackConnect *SlackConnect) Token(c context.Context, code string) (*oauth2.Token, error) {
	return slackConnect.slackOauthConfig.Exchange(c, code)
}

func (slackConnect *SlackConnect) AuthCodeURL(state string) string {
	slackConnect.log.Debugf("generating code url from slack with state = %v", state)
	return slackConnect.slackOauthConfig.AuthCodeURL(state)
}
