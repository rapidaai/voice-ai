package internal_connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	config "github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/linkedin"
)

type LinkedinConnect struct {
	ExternalConnect
	logger              commons.Logger
	linkedinOauthConfig oauth2.Config
}

var (
	LINKEDIN_AUTHENTICATION_URL   = "/auth/signin"
	LINKEDIN_AUTHENTICATION_SCOPE = []string{"openid", "profile", "email"}

	LINKEDIN_ACTION_CONNECT = "/action/linkedin"
	LINKEDIN_ACTION_SCOPE   = []string{"openid", "profile", "email"}
)

func NewLinkedinAuthenticationConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) LinkedinConnect {
	return LinkedinConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		linkedinOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), LINKEDIN_AUTHENTICATION_URL),
			ClientID:     oauthCfg.LinkedinClientId,
			ClientSecret: oauthCfg.LinkedinClientSecret,
			Scopes:       LINKEDIN_AUTHENTICATION_SCOPE,
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://www.linkedin.com/oauth/v2/authorization",
				TokenURL:  "https://www.linkedin.com/oauth/v2/accessToken",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		logger: logger,
	}
}

func NewLinkedinActionConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger) LinkedinConnect {
	return LinkedinConnect{
		linkedinOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), LINKEDIN_ACTION_CONNECT),
			ClientID:     oauthCfg.LinkedinClientId,
			ClientSecret: oauthCfg.LinkedinClientSecret,
			Scopes:       LINKEDIN_ACTION_SCOPE,
			Endpoint:     linkedin.Endpoint,
		},
		logger: logger,
	}
}

/**

Linkedin oauth
*/

func (wAuthApi *LinkedinConnect) AuthCodeURL(state string) string {
	return wAuthApi.linkedinOauthConfig.AuthCodeURL(state)
}

func (wAuthApi *LinkedinConnect) LinkedinUserInfo(c context.Context, state string, code string) (*OpenID, error) {
	if state != "linkedin" {
		wAuthApi.logger.Errorf("illegal oauth request as auth state is not matching %s %s", "linkedin", state)
		return nil, fmt.Errorf("invalid oauth state")
	}

	token, err := wAuthApi.linkedinOauthConfig.Exchange(c, code)
	if err != nil {
		wAuthApi.logger.Errorf("unable to exchange the token from linkedin %v", err)
		return nil, err
	}

	client := wAuthApi.linkedinOauthConfig.Client(c, token)
	req, err := http.NewRequest("GET", "https://api.linkedin.com/v2/userinfo", nil)
	if err != nil {
		wAuthApi.logger.Errorf("error while creating request %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	response, err := client.Do(req)
	if err != nil {
		wAuthApi.logger.Errorf("error while getting user from linkedin %v", err)
		return nil, err
	}

	defer response.Body.Close()
	// // {"email":"p_srivastav@outlook.com","email_verified":true,"family_name":"Srivastav","given_name":"Prashant","locale":{"country":"US","language":"en"},"name":"Prashant Srivastav","picture":"https://media.licdn.com/dms/image/C5603AQGslsdJ_ZIoMA/profile-displayphoto-shrink_100_100/0/1659118454695?e=1706745600\u0026v=beta\u0026t=8NmYbyO4c6gd3Y1MQjs4LZ3cmh6tYU9zc9Ghlg3FAQ0","sub":"XyBk2_14Uj"}
	var content map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&content)
	if err != nil {
		wAuthApi.logger.Errorf("unable to decode %v", err)
		return nil, err
	}
	email, ok := content["email"].(string)
	if !ok {
		return nil, errors.New("missing or invalid email")
	}

	verified, ok := content["email_verified"].(bool)
	if !ok {
		return nil, errors.New("missing or invalid email_verified")
	}

	name, ok := content["name"].(string)
	if !ok {
		return nil, errors.New("missing or invalid name")
	}

	id, ok := content["sub"].(string)
	if !ok {
		return nil, errors.New("missing or invalid sub")
	}

	return &OpenID{
		Token:    token.AccessToken,
		Source:   "linkedin",
		Email:    email,
		Verified: verified,
		Name:     name,
		Id:       id,
	}, nil
}
