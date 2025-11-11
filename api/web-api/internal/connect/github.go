package internal_connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mitchellh/mapstructure"
	config "github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GithubConnect struct {
	ExternalConnect
	logger            commons.Logger
	githubOauthConfig oauth2.Config
}

var (
	GITHUB_AUTHENTICATION_SCOPE = []string{"user"}
	GITHUB_AUTHENTICATION_URL   = "/connect-common/github" //"/auth/signin"

	GITHUB_CODE_SCOPE   = []string{"read:org"}
	GITHUB_CODE_CONNECT = "/connect-common/github"

	GITHUB_ACTION_SCOPE   = []string{}
	GITHUB_ACTION_CONNECT = "/connect-common/github"
)

func NewGithubAuthenticationConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger,
	postgres connectors.PostgresConnector) GithubConnect {
	return GithubConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		githubOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITHUB_AUTHENTICATION_URL),
			ClientID:     oauthCfg.GithubClientId,
			ClientSecret: oauthCfg.GithubClientSecret,
			Scopes:       GITHUB_AUTHENTICATION_SCOPE,
			Endpoint:     github.Endpoint,
		},
		logger: logger,
	}
}

func NewGithubCodeConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) GithubConnect {
	return GithubConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		githubOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITHUB_CODE_CONNECT),
			ClientID:     oauthCfg.GithubClientId,
			ClientSecret: oauthCfg.GithubClientSecret,
			Scopes:       GITHUB_CODE_SCOPE,
			Endpoint:     github.Endpoint,
		},
		logger: logger,
	}
}
func NewGithubActionConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) GithubConnect {
	return GithubConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		githubOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GITHUB_ACTION_CONNECT),
			ClientID:     oauthCfg.GithubClientId,
			ClientSecret: oauthCfg.GithubClientSecret,
			Scopes:       GITHUB_ACTION_SCOPE,
			Endpoint:     github.Endpoint,
		},
		logger: logger,
	}
}

func (wAuthApi *GithubConnect) AuthCodeURL(state string) string {
	return wAuthApi.githubOauthConfig.AuthCodeURL(state)

}

type githubUser struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	ID    int64  `json:"id"`
}

func (wAuthApi *GithubConnect) validateGithubUser(content map[string]interface{}) (*githubUser, error) {
	var user githubUser
	if err := mapstructure.Decode(content, &user); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub user: %w", err)
	}

	if user.ID == 0 {
		return nil, fmt.Errorf("invalid GitHub user ID")
	}

	if user.Email == "" {
		return nil, fmt.Errorf("GitHub email is required")
	}

	if user.Name == "" {
		return nil, fmt.Errorf("GitHub name is required")
	}

	return &user, nil
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
	user, err := wAuthApi.validateGithubUser(content)
	if err != nil {
		wAuthApi.logger.Errorf("unable to parse user information from github %v", err)
		return nil, err
	}

	return &OpenID{
		Token:    token.AccessToken,
		Source:   "github",
		Email:    user.Email,
		Verified: true,
		Name:     user.Name,
		Id:       strconv.FormatInt(user.ID, 10),
	}, nil
}

type GithubTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

func (gtr *GithubTokenResponse) Token() *oauth2.Token {
	// Calculate token expiry time (if ExpiresIn is provided)
	var expiry time.Time
	if gtr.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(gtr.ExpiresIn) * time.Second)
	} else {
		// GitHub tokens are often long-lived, set a far future expiry
		expiry = time.Now().Add(24 * 365 * time.Hour) // 1 year
	}
	return &oauth2.Token{
		AccessToken:  gtr.AccessToken,
		TokenType:    gtr.TokenType,
		RefreshToken: gtr.RefreshToken,
		Expiry:       expiry,
	}
}

func (gtr *GithubTokenResponse) Map() map[string]interface{} {
	var expiry time.Time
	if gtr.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(gtr.ExpiresIn) * time.Second)
	} else {
		// GitHub tokens are often long-lived, set a far future expiry
		expiry = time.Now().Add(24 * 365 * time.Hour) // 1 year
	}
	return map[string]interface{}{
		"accessToken":  gtr.AccessToken,
		"tokenType":    gtr.TokenType,
		"refreshToken": gtr.RefreshToken,
		"expiry":       expiry,
	}
}
func (wAuthApi *GithubConnect) Token(c context.Context, code string) (ExternalConnectToken, error) {
	tokenEndpoint := wAuthApi.githubOauthConfig.Endpoint.TokenURL

	// Prepare the data for the request
	data := map[string]string{
		"code":          code,
		"client_id":     wAuthApi.githubOauthConfig.ClientID,
		"client_secret": wAuthApi.githubOauthConfig.ClientSecret,
		"redirect_uri":  wAuthApi.githubOauthConfig.RedirectURL,
		"grant_type":    "authorization_code",
	}

	// Make the request
	resp, err := wAuthApi.NewHttpClient().R().
		SetContext(c).
		SetHeader("Accept", "application/json").
		SetFormData(data).
		Post(tokenEndpoint)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, errors.New("failed to exchange token, status code is not ")
	}

	// Parse the response
	var tokenResponse GithubTokenResponse
	if err := json.Unmarshal(resp.Body(), &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

type GitHubRepository struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	Description  string `json:"description"`
	HTMLURL      string `json:"html_url"`
	Organization string `json:"organization"`
}

func (githubConnect *GithubConnect) Repositories(ctx context.Context,
	token *oauth2.Token,
	q *string,
	pageToken *string) ([]GitHubRepository, error) {
	client := githubConnect.githubOauthConfig.Client(context.Background(), token)
	restyClient := githubConnect.GetClient(client)
	var results []GitHubRepository
	userRepos, err := githubConnect.fetchRepositories(restyClient, "")
	if err != nil {
		githubConnect.log.Errorf("there is no personal repository for the user or error occured %+v", err)
	}

	for _, rep := range userRepos {
		rep.Organization = "Personal"
		results = append(results, rep)
	}

	orgs, err := githubConnect.fetchUserOrganizations(restyClient)
	if err != nil {
		githubConnect.log.Errorf("there is no org associated with the user or error %+v", err)
	}

	// Fetch repositories for each organization
	for _, org := range orgs {
		repos, err := githubConnect.fetchRepositories(restyClient, org.Login)
		if err != nil {
			githubConnect.log.Errorf("there is no org repository for user or error %+v", err)
		}
		for _, rep := range repos {
			rep.Organization = org.Login
			results = append(results, rep)
		}
	}

	return results, nil
}

type GitHubOrganization struct {
	ID          int    `json:"id"`
	Login       string `json:"login"`
	AvatarURL   string `json:"avatar_url"`
	Description string `json:"description"`
}

func (githubConnect *GithubConnect) fetchUserOrganizations(client *resty.Client) ([]GitHubOrganization, error) {
	resp, err := client.R().
		SetHeader("Accept", "application/vnd.github.v3+json").
		Get("https://api.github.com/user/orgs")
	if err != nil {
		return nil, err
	}

	var orgs []GitHubOrganization
	if err := json.Unmarshal(resp.Body(), &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
}

func (githubConnect *GithubConnect) fetchRepositories(client *resty.Client, org string) ([]GitHubRepository, error) {
	var url string
	if org != "" {
		url = "https://api.github.com/orgs/" + org + "/repos"
	} else {
		url = "https://api.github.com/user/repos"
	}

	resp, err := client.R().
		SetHeader("Accept", "application/vnd.github.v3+json").
		Get(url)
	if err != nil {
		return nil, err
	}

	var repos []GitHubRepository
	if err := json.Unmarshal(resp.Body(), &repos); err != nil {
		return nil, err
	}

	return repos, nil
}
