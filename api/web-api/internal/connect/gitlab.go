package internal_connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	config "github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/gitlab"
)

type GitlabConnect struct {
	ExternalConnect
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

func NewGitlabAuthenticationConnect(cfg *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector) GitlabConnect {
	return GitlabConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
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

func NewGitlabCodeConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) GitlabConnect {
	return GitlabConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		gitlabOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITLAB_CODE_CONNECT),
			ClientID:     oauthCfg.GitlabClientId,
			ClientSecret: oauthCfg.GitlabClientSecret,
			Scopes:       GITLAB_CODE_SCOPE,
			Endpoint:     gitlab.Endpoint,
		},
		logger: logger,
	}
}
func NewGitlabActionConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) GitlabConnect {
	return GitlabConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		gitlabOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITLAB_ACTION_CONNECT),
			ClientID:     oauthCfg.GitlabClientId,
			ClientSecret: oauthCfg.GitlabClientSecret,
			Scopes:       GITLAB_ACTION_SCOPE,
			Endpoint:     gitlab.Endpoint,
		},
		logger: logger,
	}
}

func (gitlabConnect *GitlabConnect) AuthCodeURL(state string) string {
	return gitlabConnect.gitlabOauthConfig.AuthCodeURL(state)
}

type GitlabTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

func (gtr *GitlabTokenResponse) Token() *oauth2.Token {
	// Calculate token expiry time (if ExpiresIn is provided)
	var expiry time.Time
	if gtr.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(gtr.ExpiresIn) * time.Second)
	} else {
		// GitLab tokens may have an unspecified expiry, set a far future expiry
		expiry = time.Now().Add(24 * 365 * time.Hour) // 1 year
	}

	return &oauth2.Token{
		AccessToken:  gtr.AccessToken,
		TokenType:    gtr.TokenType,
		RefreshToken: gtr.RefreshToken,
		Expiry:       expiry,
	}
}

func (gtr *GitlabTokenResponse) Map() map[string]interface{} {
	var expiry time.Time
	if gtr.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(gtr.ExpiresIn) * time.Second)
	} else {
		// GitLab tokens may have an unspecified expiry, set a far future expiry
		expiry = time.Now().Add(24 * 365 * time.Hour) // 1 year
	}
	return map[string]interface{}{
		"accessToken":  gtr.AccessToken,
		"tokenType":    gtr.TokenType,
		"refreshToken": gtr.RefreshToken,
		"expiry":       expiry,
	}
}

func (gitlabConnect *GitlabConnect) Token(c context.Context, code string) (ExternalConnectToken, error) {
	// return gitlabConnect.gitlabOauthConfig.Exchange(c, code)
	tokenEndpoint := gitlabConnect.gitlabOauthConfig.Endpoint.TokenURL

	// Prepare the data for the request
	data := map[string]string{
		"code":          code,
		"client_id":     gitlabConnect.gitlabOauthConfig.ClientID,
		"client_secret": gitlabConnect.gitlabOauthConfig.ClientSecret,
		"redirect_uri":  gitlabConnect.gitlabOauthConfig.RedirectURL,
		"grant_type":    "authorization_code",
	}

	// Make the request
	resp, err := gitlabConnect.NewHttpClient().R().
		SetContext(c).
		SetHeader("Accept", "application/json").
		SetFormData(data).
		Post(tokenEndpoint)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, errors.New("failed to exchange token")
	}

	// Parse the response
	var tokenResponse GitlabTokenResponse
	if err := json.Unmarshal(resp.Body(), &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}
