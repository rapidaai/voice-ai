package internal_connects

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleConnect struct {
	logger            commons.Logger
	googleOauthConfig oauth2.Config
}

var (
	GOOGLE_AUTHENTICATION_STATE = "google"
	GOOGLE_AUTHENTICATION_SCOPE = []string{"email", "profile"}
	GOOGLE_AUTHENTICATION_URL   = "/auth/signin"

	// GOOGLE_DRIVE_STATE       = "connect/"
	GOOGLE_DRIVE_SCOPE = []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/drive.readonly",
		"https://www.googleapis.com/auth/drive.file",
		"https://www.googleapis.com/auth/drive.metadata.readonly",
	}
	GOOGLE_DRIVE_CONNECT_URL = "/connect-knowledge/google-drive"

	GOOGLE_GMAIL_SCOPE = []string{
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/gmail.compose",
		"https://www.googleapis.com/auth/gmail.send",
	}
	GOOGLE_GMAIL_CONNECT_URL = "/connect-action/gmail"
)

func NewGoogleAuthenticationConnect(cfg *config.AppConfig, logger commons.Logger) GoogleConnect {
	return GoogleConnect{
		googleOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GOOGLE_AUTHENTICATION_URL),
			ClientID:     cfg.GoogleClientId,
			ClientSecret: cfg.GoogleClientSecret,
			Scopes:       GOOGLE_AUTHENTICATION_SCOPE,
			Endpoint:     google.Endpoint,
		},
		logger: logger,
	}
}

func NewGoogleDriveConnect(cfg *config.AppConfig, logger commons.Logger) GoogleConnect {
	return GoogleConnect{
		googleOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GOOGLE_DRIVE_CONNECT_URL),
			ClientID:     cfg.GoogleClientId,
			ClientSecret: cfg.GoogleClientSecret,
			Scopes:       GOOGLE_DRIVE_SCOPE,
			Endpoint:     google.Endpoint,
		},
		logger: logger,
	}
}

func NewGmailConnect(cfg *config.AppConfig, logger commons.Logger) GoogleConnect {
	return GoogleConnect{
		googleOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GOOGLE_GMAIL_CONNECT_URL),
			ClientID:     cfg.GoogleClientId,
			ClientSecret: cfg.GoogleClientSecret,
			Scopes:       GOOGLE_GMAIL_SCOPE,
			Endpoint:     google.Endpoint,
		},
		logger: logger,
	}
}

/*
*

Google implimentation
*/

// "google"
func (wAuthApi *GoogleConnect) AuthCodeURL(state string) string {
	return wAuthApi.googleOauthConfig.AuthCodeURL(state)
}
func (wAuthApi *GoogleConnect) GoogleUserInfo(c context.Context, state string, code string) (*OpenID, error) {
	if state != "google" {
		wAuthApi.logger.Errorf("illegal oauth request as auth state is not matching %s %s", "google", state)
		return nil, fmt.Errorf("invalid oauth state")
	}
	token, err := wAuthApi.googleOauthConfig.Exchange(c, code)
	if err != nil {
		wAuthApi.logger.Errorf("google authentication exchange failed %v", err)
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		wAuthApi.logger.Errorf("unable to get userinfo using the access token %v", err)
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()

	var content OpenID
	err = json.NewDecoder(response.Body).Decode(&content)
	content.Source = "google"
	content.Token = token.AccessToken
	if err != nil {
		wAuthApi.logger.Errorf("unable to decode the response body of the user info %v", err)
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}
	return &content, nil
}
