package internal_connect

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	config "github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"golang.org/x/oauth2"
)

type HubspotConnect struct {
	ExternalConnect
	logger             commons.Logger
	hubspotOauthConfig oauth2.Config
}

var (
	HUBSPOT_SCOPE = []string{"crm.objects.leads.read", "crm.objects.leads.write",
		"crm.objects.companies.read", "crm.objects.companies.write",
		"crm.objects.contacts.read", "crm.objects.contacts.write"}
	HUBSPOT_CONNECT = "/connect-crm/hubspot"
)

func NewHubspotConnect(cfg *config.WebAppConfig, oAuthcfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) HubspotConnect {
	return HubspotConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		hubspotOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), HUBSPOT_CONNECT),
			ClientID:     oAuthcfg.HubspotClientId,
			ClientSecret: oAuthcfg.HubspotClientSecret,
			Scopes:       HUBSPOT_SCOPE,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://app.hubspot.com/oauth/authorize",
				TokenURL: "https://api.hubapi.com/oauth/v1/token",
			},
		},
		logger: logger,
	}
}

type HubspotTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (gtr *HubspotTokenResponse) Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  gtr.AccessToken,
		RefreshToken: gtr.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(gtr.ExpiresIn) * time.Second),
	}
}

func (gtr *HubspotTokenResponse) Map() map[string]interface{} {
	return map[string]interface{}{
		"accessToken":  gtr.AccessToken,
		"refreshToken": gtr.RefreshToken,
		"expiresIn":    gtr.ExpiresIn,
	}
}

func (hubspotConnect *HubspotConnect) Token(c context.Context, code string) (ExternalConnectToken, error) {
	resp, err := hubspotConnect.NewHttpClient().R().
		SetBasicAuth(hubspotConnect.hubspotOauthConfig.ClientID, hubspotConnect.hubspotOauthConfig.ClientSecret).
		SetHeader("content-type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"client_id":     hubspotConnect.hubspotOauthConfig.ClientID,
			"client_secret": hubspotConnect.hubspotOauthConfig.ClientSecret,
			"redirect_uri":  hubspotConnect.hubspotOauthConfig.RedirectURL,
		}).
		Post(hubspotConnect.hubspotOauthConfig.Endpoint.TokenURL)
	if err != nil {
		hubspotConnect.log.Errorf("Error while creating request: %v", err)
		return nil, err
	}

	if resp.IsError() {
		hubspotConnect.log.Errorf("Error response: %s", resp.String())
		return nil, fmt.Errorf("failed to get token: %s", resp.Status())
	}

	var tokenResponse HubspotTokenResponse
	err = json.Unmarshal(resp.Body(), &tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token response: %v", err)
	}

	return &tokenResponse, nil
}

func (hubspotConnect *HubspotConnect) RefreshToken(c context.Context, token *oauth2.Token) (ExternalConnectToken, error) {
	resp, err := hubspotConnect.NewHttpClient().R().
		SetBasicAuth(hubspotConnect.hubspotOauthConfig.ClientID, hubspotConnect.hubspotOauthConfig.ClientSecret).
		SetHeader("content-type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": token.RefreshToken,
			"client_id":     hubspotConnect.hubspotOauthConfig.ClientID,
			"client_secret": hubspotConnect.hubspotOauthConfig.ClientSecret,
		}).
		Post(hubspotConnect.hubspotOauthConfig.Endpoint.TokenURL)
	if err != nil {
		hubspotConnect.log.Errorf("Error while creating request: %v", err)
		return nil, err
	}

	if resp.IsError() {
		hubspotConnect.log.Errorf("Error response: %s", resp.String())
		return nil, fmt.Errorf("failed to get token: %s", resp.Status())
	}

	var tokenResponse HubspotTokenResponse
	err = json.Unmarshal(resp.Body(), &tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token response: %v", err)
	}
	return &tokenResponse, nil
}

func (hubspotConnect *HubspotConnect) AuthCodeURL(state string) string {
	hubspotConnect.log.Debugf("generating code url from notion with state = %v", state)
	return hubspotConnect.hubspotOauthConfig.AuthCodeURL(state)
}
