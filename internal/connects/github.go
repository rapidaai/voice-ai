package internal_connects

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GithubConnect struct {
	logger            commons.Logger
	githubOauthConfig oauth2.Config
}

var (
	GITHUB_AUTHENTICATION_SCOPE = []string{"user"}
	GITHUB_AUTHENTICATION_URL   = "/connect-common/github" //"/auth/signin"

	GITHUB_CODE_SCOPE   = []string{}
	GITHUB_CODE_CONNECT = "/connect-common/github"

	GITHUB_ACTION_SCOPE   = []string{}
	GITHUB_ACTION_CONNECT = "/connect-common/github"
)

func NewGithubAuthenticationConnect(cfg *config.AppConfig, logger commons.Logger) GithubConnect {
	return GithubConnect{
		githubOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITHUB_AUTHENTICATION_URL),
			ClientID:     cfg.GithubClientId,
			ClientSecret: cfg.GithubClientSecret,
			Scopes:       GITHUB_AUTHENTICATION_SCOPE,
			Endpoint:     github.Endpoint,
		},
		logger: logger,
	}
}

func NewGithubCodeConnect(cfg *config.AppConfig, logger commons.Logger) GithubConnect {
	return GithubConnect{
		githubOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITHUB_CODE_CONNECT),
			ClientID:     cfg.GithubClientId,
			ClientSecret: cfg.GithubClientSecret,
			Scopes:       GITHUB_CODE_SCOPE,
			Endpoint:     github.Endpoint,
		},
		logger: logger,
	}
}
func NewGithubActionConnect(cfg *config.AppConfig, logger commons.Logger) GithubConnect {
	return GithubConnect{
		githubOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITHUB_ACTION_CONNECT),
			ClientID:     cfg.GithubClientId,
			ClientSecret: cfg.GithubClientSecret,
			Scopes:       GITHUB_ACTION_SCOPE,
			Endpoint:     github.Endpoint,
		},
		logger: logger,
	}
}

func (wAuthApi *GithubConnect) AuthCodeURL(state string) string {
	return wAuthApi.githubOauthConfig.AuthCodeURL(state)

}

func (wAuthApi *GithubConnect) GithubUserInfo(c context.Context, state string, code string) (*OpenID, error) {
	if state != "github" {
		wAuthApi.logger.Errorf("illegal oauth request as auth state is not matching %s %s", "github", state)
		return nil, fmt.Errorf("invalid oauth state")
	}

	token, err := wAuthApi.githubOauthConfig.Exchange(c, code)
	if err != nil {
		wAuthApi.logger.Errorf("unable to exchange the token from github %v", err)
		return nil, err
	}

	oauthClient := wAuthApi.githubOauthConfig.Client(c, token)
	req, err := http.NewRequest("POST", "https://api.github.com/user", nil)
	if err != nil {
		wAuthApi.logger.Errorf("error while creating request %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	response, err := oauthClient.Do(req)
	if err != nil {
		wAuthApi.logger.Errorf("error while getting user from linkedin %v", err)
		return nil, err
	}

	defer response.Body.Close()
	var content map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&content)
	if err != nil {
		wAuthApi.logger.Errorf("unable to decode %v", err)
		return nil, err
	}
	return &OpenID{
		Token: token.AccessToken, Source: "github",
		Email:    content["email"].(string),
		Verified: true,
		Name:     content["name"].(string),
		Id:       fmt.Sprintf("%f", content["id"].(float64)),
	}, nil
}
