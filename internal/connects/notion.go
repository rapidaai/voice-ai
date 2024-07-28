package internal_connects

import (
	"context"
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"golang.org/x/oauth2"
)

type NotionConnect struct {
	ExternalConnect
	logger            commons.Logger
	notionOauthConfig oauth2.Config
}

var (

	// MICROSOFT_DRIVE_STATE       = "connect/"
	NOTION_WORKPLACE_SCOPE = []string{"read_content"}
	// Scopes:
	NOTION_WORKPLACE_CONNECT = "/connect-knowledge/notion"
)

func NewNotionWorkplaceConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) NotionConnect {
	return NotionConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		notionOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), NOTION_WORKPLACE_CONNECT),
			ClientID:     cfg.NotionClientId,
			ClientSecret: cfg.NotionClientSecret,
			Scopes:       NOTION_WORKPLACE_SCOPE,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://api.notion.com/v1/oauth/authorize",
				TokenURL: "https://api.notion.com/v1/oauth/token",
			},
		},
		logger: logger,
	}
}

func (notionConnect *NotionConnect) Token(c context.Context, code string) (*oauth2.Token, error) {
	return notionConnect.notionOauthConfig.Exchange(c, code)
}

func (notionConnect *NotionConnect) AuthCodeURL(state string) string {
	notionConnect.log.Debugf("generating code url from notion with state = %v", state)
	return notionConnect.notionOauthConfig.AuthCodeURL(state)
}

// Define data structures for parsing API responses
type NotionPage struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	URL         string       `json:"url"`
	NotionSpace *NotionSpace `json:"space"`
}

type NotionPages struct {
	Results       []*NotionPage `json:"results,omitempty"`
	NextPageToken string        `json:"next_cursor,omitempty"`
}

type NotionSpace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type NotionSpaces struct {
	Results []*NotionSpace `json:"results,omitempty"`
}

// give all the pages in side. a space
func (notionConnect *NotionConnect) NotionPages(ctx context.Context,
	token *oauth2.Token,
	q *string,
	pageToken *string) (*NotionPages, error) {
	client := notionConnect.notionOauthConfig.Client(context.Background(), token)
	restyClient := resty.NewWithClient(client)

	// Fetch Notion spaces
	spaces, err := notionConnect.fetchSpaces(restyClient)
	if err != nil {
		log.Fatalf("Error fetching spaces: %v", err)
	}

	cfp := &NotionPages{}
	for _, space := range spaces.Results {
		fmt.Printf("Space ID: %s, Space Name: %s, Space URL: %s\n", space.ID, space.Name, space.URL)
		// Fetch pages for each space
		pages, err := notionConnect.fetchPages(restyClient, space.ID)
		if err != nil {
			log.Fatalf("Error fetching pages for space %s: %v", space.ID, err)
		}
		for _, page := range pages.Results {
			// fmt.Printf("Page ID: %s, Page Title: %s, Page URL: %s\n", page.ID, page.Title, page.URL)
			page.NotionSpace = space
			cfp.Results = append(cfp.Results, page)
		}
	}
	return cfp, nil
}
func (notionConnect *NotionConnect) fetchSpaces(client *resty.Client) (*NotionSpaces, error) {
	var spaces NotionSpaces

	// Example Notion API URL for retrieving spaces (workspaces)
	apiURL := "https://api.notion.com/v1/users/{user_id}/workspaces"

	_, err := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetResult(&spaces).
		Get(apiURL)

	if err != nil {
		return nil, fmt.Errorf("error fetching spaces: %v", err)
	}

	return &spaces, nil
}

func (notionConnect *NotionConnect) fetchPages(client *resty.Client, spaceID string) (*NotionPages, error) {
	var pages NotionPages

	apiURL := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", spaceID)
	_, err := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetResult(&pages).
		Post(apiURL)

	if err != nil {
		return nil, fmt.Errorf("error fetching pages: %v", err)
	}

	return &pages, nil
}
