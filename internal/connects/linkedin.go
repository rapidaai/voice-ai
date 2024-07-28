package internal_connects

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/linkedin"
)

type LinkedinConnect struct {
	logger              commons.Logger
	linkedinOauthConfig oauth2.Config
}

var (
	LINKEDIN_AUTHENTICATION_URL   = "/auth/signin"
	LINKEDIN_AUTHENTICATION_SCOPE = []string{"openid", "profile", "email"}

	LINKEDIN_ACTION_CONNECT = "/action/linkedin"
	LINKEDIN_ACTION_SCOPE   = []string{"openid", "profile", "email"}
)

func NewLinkedinAuthenticationConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) LinkedinConnect {
	return LinkedinConnect{
		linkedinOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), LINKEDIN_AUTHENTICATION_URL),
			ClientID:     cfg.LinkedinClientId,
			ClientSecret: cfg.LinkedinClientSecret,
			Scopes:       LINKEDIN_AUTHENTICATION_SCOPE,
			Endpoint:     linkedin.Endpoint,
		},
		logger: logger,
	}
}

func NewLinkedinActionConnect(cfg *config.AppConfig, logger commons.Logger) LinkedinConnect {
	return LinkedinConnect{
		linkedinOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), LINKEDIN_ACTION_CONNECT),
			ClientID:     cfg.LinkedinClientId,
			ClientSecret: cfg.LinkedinClientSecret,
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
	return &OpenID{
		Token: token.AccessToken, Source: "linkedin",
		Email:    content["email"].(string),
		Verified: content["email_verified"].(bool),
		Name:     content["name"].(string),
		Id:       content["sub"].(string),
	}, nil
}
