package internal_connect

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	config "github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
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

func NewNotionWorkplaceConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) NotionConnect {
	return NotionConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		notionOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), NOTION_WORKPLACE_CONNECT),
			ClientID:     oauthCfg.NotionClientId,
			ClientSecret: oauthCfg.NotionClientSecret,
			Scopes:       NOTION_WORKPLACE_SCOPE,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://api.notion.com/v1/oauth/authorize",
				TokenURL: "https://api.notion.com/v1/oauth/token",
			},
		},
		logger: logger,
	}
}

type NotionTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	BotId       string `json:"bot_id"`
	Owner       struct {
		Type string `json:"type"`
		User struct {
			Object string `json:"object"`
			Id     string `json:"id"`
		} `json:"user"`
	} `json:"owner"`
	WorkspaceName string `json:"workspace_name"`
	WorkspaceIcon string `json:"workspace_icon"`
	WorkspaceId   string `json:"workspace_id"`
	RequestId     string `json:"request_id"`
}

func (gtr *NotionTokenResponse) Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken: gtr.AccessToken,
		TokenType:   gtr.TokenType,
	}
}

func (gtr *NotionTokenResponse) Map() map[string]interface{} {
	return map[string]interface{}{
		"accessToken":   gtr.AccessToken,
		"tokenType":     gtr.TokenType,
		"botId":         gtr.BotId,
		"owner":         gtr.Owner,
		"workspaceName": gtr.WorkspaceName,
		"workspaceIcon": gtr.WorkspaceIcon,
		"workspaceId":   gtr.WorkspaceId,
		"requestId":     gtr.RequestId,
	}
}

func (notionConnect *NotionConnect) Token(c context.Context, code string) (ExternalConnectToken, error) {
	resp, err := notionConnect.NewHttpClient().R().
		SetBasicAuth(notionConnect.notionOauthConfig.ClientID, notionConnect.notionOauthConfig.ClientSecret).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"grant_type":   "authorization_code",
			"code":         code,
			"redirect_uri": notionConnect.notionOauthConfig.RedirectURL,
		}).
		Post(notionConnect.notionOauthConfig.Endpoint.TokenURL)
	if err != nil {
		notionConnect.log.Errorf("Error while creating request: %v", err)
		return nil, err
	}

	if resp.IsError() {
		notionConnect.log.Errorf("Error response: %s", resp.String())
		return nil, fmt.Errorf("failed to get token: %s", resp.Status())
	}

	var tokenResponse NotionTokenResponse
	err = json.Unmarshal(resp.Body(), &tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token response: %v", err)
	}

	return &tokenResponse, nil
}

func (notionConnect *NotionConnect) AuthCodeURL(state string) string {
	notionConnect.log.Debugf("generating code url from notion with state = %v", state)
	return notionConnect.notionOauthConfig.AuthCodeURL(state)
}

// give all the pages in side. a space
func (notionConnect *NotionConnect) NotionPages(ctx context.Context,
	token *oauth2.Token,
	q *string,
	pageToken *string) (*NotionSearchResult, error) {

	client := notionConnect.notionOauthConfig.Client(context.Background(), token)
	restyClient := notionConnect.GetClient(client)

	// Fetch Notion spaces
	spaces, err := notionConnect.fetchAllContent(restyClient)
	if err != nil {
		notionConnect.log.Errorf("Error fetching spaces: %v", err)
		return nil, err
	}

	return spaces, nil
}

// Define the structure for rich text objects
type NotionRichText struct {
	Type      string     `json:"type"`
	Text      NotionText `json:"text"`
	PlainText string     `json:"plain_text"`
	Href      string     `json:"href"`
}

type NotionText struct {
	Content string      `json:"content"`
	Link    interface{} `json:"link"`
}

type NotionSearchResult struct {
	Object     string         `json:"object"`
	Results    []NotionResult `json:"results"`
	NextCursor string         `json:"next_cursor"`
	HasMore    bool           `json:"has_more"`
}

type NotionResult struct {
	Object         string                 `json:"object"`
	ID             string                 `json:"id"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
	Properties     map[string]interface{} `json:"properties"`
	Title          []NotionRichText       `json:"title"`
	URL            string                 `json:"url"`
	TitleStr       string                 `json:"title_str"`
}

func (notionConnect *NotionConnect) fetchAllContent(client *resty.Client) (*NotionSearchResult, error) {
	var databases NotionSearchResult
	apiURL := "https://api.notion.com/v1/search"
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Notion-Version", "2022-06-28").
		SetBody(map[string]interface{}{
			"sort": map[string]string{
				"direction": "descending",
				"timestamp": "last_edited_time",
			}}).
		SetResult(&databases).
		Post(apiURL)
	if err != nil {
		notionConnect.log.Errorf("Error fetching database: %v", err)
		return nil, fmt.Errorf("error fetching databases: %v", err)
	}
	if resp.IsError() {
		notionConnect.log.Errorf("Error fetching database: %v", err)
		return nil, fmt.Errorf("failed to fetch databases: %s", resp.Status())
	}

	for i := range databases.Results {
		if databases.Results[i].Object == "page" {
			if titleProperty, ok := databases.Results[i].Properties["title"].(map[string]interface{}); ok {
				if titleArray, ok := titleProperty["title"].([]interface{}); ok {
					title := ""
					for _, t := range titleArray {
						if text, ok := t.(map[string]interface{}); ok {
							title += text["plain_text"].(string)
						}
					}

					databases.Results[i].TitleStr = title
				}
			}
			continue
		}
		databases.Results[i].TitleStr = notionConnect.getTitleString(databases.Results[i].Title)
	}

	return &databases, nil
}

func (notionConnect *NotionConnect) getTitleString(titleRichText []NotionRichText) string {
	notionConnect.log.Debugf("Giving you the text title form %+v", titleRichText)
	var title string
	for _, text := range titleRichText {
		title += text.PlainText
	}
	return title
}
