package internal_connect

import (
	"context"
	"fmt"

	config "github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
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

func NewSlackActionConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) SlackConnect {
	return SlackConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		slackOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", "https://7c07-2401-4900-1cbd-4c13-3408-70d2-3232-43f2.ngrok-free.app", SLACK_SEND_MESSAGE_CONNECT),
			ClientID:     oauthCfg.SlackClientId,
			ClientSecret: oauthCfg.SlackClientSecret,
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
