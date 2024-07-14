package internal_connects

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type MicrosoftConnect struct {
	logger               commons.Logger
	microsoftOauthConfig oauth2.Config
}

var (
	MICROSOFT_AUTHENTICATION_STATE = "microsoft"
	MICROSOFT_AUTHENTICATION_SCOPE = []string{
		"https://graph.microsoft.com/.default",
	}
	MICROSOFT_AUTHENTICATION_URL = "/auth/signin"

	// MICROSOFT_DRIVE_STATE       = "connect/"
	MICROSOFT_ONEDRIVE_SCOPE = []string{
		"Files.Read",
		"Files.Read.All",
		"Files.ReadWrite",
		"Files.ReadWrite.All",
	}
	MICROSOFT_ONEDRIVE_CONNECT = "/connect-knowledge/one-drive"

	MICROSOFT_SHAREPOINT_SCOPE = []string{
		"Sites.Read",
		"Sites.Read.All",
		"Files.ReadWrite",
		"Sites.ReadWrite.All",
		"Files.Read",
		"Files.Read.All",
		"Files.ReadWrite",
		"Files.ReadWrite.All",
	}
	MICROSOFT_SHAREPOINT_CONNECT = "/connect-knowledge/share-point"
)

func NewMicrosoftAuthenticationConnect(cfg *config.AppConfig, logger commons.Logger) MicrosoftConnect {
	return MicrosoftConnect{
		microsoftOauthConfig: oauth2.Config{
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
		microsoftOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), MICROSOFT_SHAREPOINT_CONNECT),
			ClientID:     cfg.MicrosoftClientId,
			ClientSecret: cfg.MicrosoftClientSecret,
			Scopes:       MICROSOFT_SHAREPOINT_SCOPE,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			},
		},
		logger: logger,
	}
}

func NewMicrosoftOnedriveConnect(cfg *config.AppConfig, logger commons.Logger) MicrosoftConnect {
	return MicrosoftConnect{
		microsoftOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), MICROSOFT_ONEDRIVE_CONNECT),
			ClientID:     cfg.MicrosoftClientId,
			ClientSecret: cfg.MicrosoftClientSecret,
			Scopes:       MICROSOFT_ONEDRIVE_SCOPE,
			Endpoint:     microsoft.LiveConnectEndpoint,
		},
		logger: logger,
	}
}

func (microsoft *MicrosoftConnect) codeVerifier() string {
	verifier := uuid.New().String()
	return base64.RawURLEncoding.EncodeToString([]byte(verifier))
}

func (microsoft *MicrosoftConnect) codeChallenge(verifier string) string {
	hash := sha256.New()
	hash.Write([]byte(verifier))
	sha := hash.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sha)
}

func (microsoft *MicrosoftConnect) AuthCodeURL(state string) string {
	codeChallenge := microsoft.codeChallenge(microsoft.codeVerifier())
	return microsoft.microsoftOauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("code_challenge", codeChallenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}
