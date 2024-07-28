package internal_connects

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"golang.org/x/oauth2"
)

type AtlassianConnect struct {
	ExternalConnect
	logger               commons.Logger
	atlassianOauthConfig oauth2.Config
}

var (
	CONFLUENCE_CONNECT_URL = "/connect-common/atlassian"
	CONFLUENCE_SCOPE       = [...]string{
		"search:confluence",
		"read:space:confluence",
		"read:confluence-content.summary",
		"read:confluence-content.all"}

	JIRA_SCOPE       = [...]string{}
	JIRA_CONNECT_URL = "/connect-common/atlassian"
)

func NewConfluenceConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) AtlassianConnect {
	return AtlassianConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		atlassianOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), CONFLUENCE_CONNECT_URL),
			ClientID:     cfg.AtlassianClientId,
			ClientSecret: cfg.AtlassianClientSecret,
			Scopes:       CONFLUENCE_SCOPE[:],
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://auth.atlassian.com/authorize",
				TokenURL:  "https://auth.atlassian.com/oauth/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		logger: logger,
	}
}

func NewJiraConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) AtlassianConnect {
	return AtlassianConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		atlassianOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), JIRA_CONNECT_URL),
			ClientID:     cfg.AtlassianClientId,
			ClientSecret: cfg.AtlassianClientSecret,
			Scopes:       JIRA_SCOPE[:],
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://auth.atlassian.com/authorize",
				TokenURL:  "https://auth.atlassian.com/oauth/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		logger: logger,
	}
}

// https://auth.atlassian.com/authorize?audience=api.atlassian.com&client_id=Et8qcoSIpSs1h1MMoRgU0rgbU9vftbCo&scope=write%3Aconfluence-content%20write%3Aconfluence-file%20readonly%3Acontent.attachment%3Aconfluence%20write%3Aconfluence-groups%20search%3Aconfluence%20read%3Aconfluence-content.summary%20read%3Aconfluence-content.all&redirect_uri=https%3A%2F%2Frapida.ai%2Fconnect%2Fatlassian&state=${YOUR_USER_BOUND_VALUE}&response_type=code&prompt=consent

func (atlassianConnect *AtlassianConnect) AuthCodeURL(state string) string {
	return atlassianConnect.atlassianOauthConfig.AuthCodeURL(state)
}

func (atlassianConnect *AtlassianConnect) Token(c context.Context, code string) (*oauth2.Token, error) {
	return atlassianConnect.atlassianOauthConfig.Exchange(c, code)
}

// give all the pages in side. a space
func (atlassianConnect *AtlassianConnect) ConfluencePages(ctx context.Context,
	token *oauth2.Token,
	q *string,
	pageToken *string) (*ConfluencePages, error) {

	client := atlassianConnect.atlassianOauthConfig.Client(ctx, token)
	restyClient := resty.NewWithClient(client)

	resourceUrl, err := atlassianConnect.fetchConfluenceBaseURL(restyClient)
	if err != nil {
		atlassianConnect.log.Errorf("Unable to get resource url from confluence +%v", err)
		return nil, err
	}
	spaces, err := atlassianConnect.fetchSpaces(restyClient, resourceUrl)
	if err != nil {
		atlassianConnect.log.Errorf("Error fetching spaces: %v", err)
		return nil, err
	}
	cfp := &ConfluencePages{}
	for _, space := range spaces {
		atlassianConnect.log.Debugf("Space Key: %s, Space Name: %s\n", space.Key, space.Name)
		pages, err := atlassianConnect.fetchPages(restyClient, resourceUrl, space.Key)
		if err != nil {
			log.Fatalf("Error fetching pages for space %s: %v", space.Key, err)
		}
		for _, page := range pages.Items {
			atlassianConnect.log.Debugf("Page ID: %s, Page Title: %s, Page URL: %s\n", page.ID, page.Title, page.URL)
			page.ConfluenceSpace = space
			cfp.Items = append(cfp.Items, page)
		}
	}
	return cfp, nil
}

type ConfluencePage struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	URL             string          `json:"url"`
	ConfluenceSpace ConfluenceSpace `json:"space"`
}

type ConfluencePages struct {
	Items         []*ConfluencePage `json:"items,omitempty"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
}

// Define data structures for parsing API responses
type ConfluenceAccessibleResource struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type ConfluenceSpace struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type ConfluenceSpaceResponse struct {
	Results []ConfluenceSpace `json:"results"`
}

// getting all the workspace of confluence which user gave the permission to
func (atlassianConnect *AtlassianConnect) fetchConfluenceBaseURL(client *resty.Client) (string, error) {

	var resources []ConfluenceAccessibleResource
	_, err := client.R().
		SetHeader("Accept", "application/json").
		SetResult(&resources).
		Get("https://api.atlassian.com/oauth/token/accessible-resources")

	if err != nil {
		return "", fmt.Errorf("error fetching accessible resources: %v", err)
	}

	if len(resources) == 0 {
		return "", fmt.Errorf("no accessible resources found")
	}

	if len(resources) > 1 {
		atlassianConnect.log.Warnf("there are multiple resouce url recieved for the user, Will have to do this in future.")
	}

	// Assuming the first resource is the Confluence instance you want to access
	return resources[0].URL, nil
}

func (atlassianConnect *AtlassianConnect) fetchSpaces(client *resty.Client, baseURL string) ([]ConfluenceSpace, error) {
	resp, err := client.R().
		SetQueryParams(map[string]string{
			"limit": "100",
		}).
		Get(fmt.Sprintf("%s/rest/api/space", baseURL))

	if err != nil {
		return nil, fmt.Errorf("error fetching spaces: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected response code: %d", resp.StatusCode())
	}

	var spaceResp ConfluenceSpaceResponse
	if err := json.Unmarshal(resp.Body(), &spaceResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling space response: %v", err)
	}
	return spaceResp.Results, nil
}

func (atlassianConnect *AtlassianConnect) fetchPages(client *resty.Client, baseURL, spaceKey string) (*ConfluencePages, error) {
	var pages ConfluencePages
	_, err := client.R().
		SetQueryParams(map[string]string{
			"spaceKey": spaceKey,
			"limit":    "50", // Set the number of items per page
		}).
		SetResult(&pages).
		Get(fmt.Sprintf("%s/rest/api/content", baseURL))

	if err != nil {
		return nil, fmt.Errorf("error fetching pages: %v", err)
	}

	return &pages, nil
}
