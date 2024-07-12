package internal_connects

import (
	"fmt"

	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type MicrosoftConnect struct {
	logger              commons.Logger
	linkedinOauthConfig oauth2.Config
}

var (
	MICROSOFT_AUTHENTICATION_STATE = "microsoft"
	MICROSOFT_AUTHENTICATION_SCOPE = []string{"email", "profile"}
	MICROSOFT_AUTHENTICATION_URL   = "/auth/signin"

	// MICROSOFT_DRIVE_STATE       = "connect/"
	MICROSOFT_ONEDRIVE_SCOPE   = []string{"email", "profile"}
	MICROSOFT_ONEDRIVE_CONNECT = "/connect/one-drive"

	MICROSOFT_SHAREPOINT_SCOPE   = []string{"email", "profile"}
	MICROSOFT_SHAREPOINT_CONNECT = "/connect/share-point"
)

func NewMicrosoftAuthenticationConnect(cfg *config.AppConfig, logger commons.Logger) MicrosoftConnect {
	return MicrosoftConnect{
		linkedinOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), MICROSOFT_AUTHENTICATION_URL),
			ClientID:     cfg.MicrosoftClientId,
			ClientSecret: cfg.MicrosoftClientSecret,
			Scopes:       MICROSOFT_AUTHENTICATION_SCOPE,
			Endpoint:     microsoft.LiveConnectEndpoint,
		},
		logger: logger,
	}
}

func NewMicrosoftSharepointConnect(cfg *config.AppConfig, logger commons.Logger) MicrosoftConnect {
	return MicrosoftConnect{
		linkedinOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), MICROSOFT_SHAREPOINT_CONNECT),
			ClientID:     cfg.MicrosoftClientId,
			ClientSecret: cfg.MicrosoftClientSecret,
			Scopes:       MICROSOFT_SHAREPOINT_SCOPE,
			Endpoint:     microsoft.LiveConnectEndpoint,
		},
		logger: logger,
	}
}

func NewMicrosoftOnedriveConnect(cfg *config.AppConfig, logger commons.Logger) MicrosoftConnect {
	return MicrosoftConnect{
		linkedinOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), MICROSOFT_ONEDRIVE_CONNECT),
			ClientID:     cfg.MicrosoftClientId,
			ClientSecret: cfg.MicrosoftClientSecret,
			Scopes:       MICROSOFT_ONEDRIVE_SCOPE,
			Endpoint:     microsoft.LiveConnectEndpoint,
		},
		logger: logger,
	}
}
