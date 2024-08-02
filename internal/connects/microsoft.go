package internal_connects

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type MicrosoftConnect struct {
	ExternalConnect
	logger               commons.Logger
	microsoftOauthConfig oauth2.Config
}

var (
	MICROSOFT_AUTHENTICATION_STATE = "microsoft"
	MICROSOFT_AUTHENTICATION_SCOPE = []string{
		"https://graph.microsoft.com/.default",
	}
	MICROSOFT_AUTHENTICATION_URL = "/auth/signin"

	// MICROSOFT_DRIVE_STATE       = "connect/"
	MICROSOFT_ONEDRIVE_SCOPE = []string{
		"Files.Read",
		"Files.Read.All",
		"offline_access",
		"Files.ReadWrite",
		"Files.ReadWrite.All",
	}
	MICROSOFT_ONEDRIVE_CONNECT = "/connect-knowledge/one-drive"

	MICROSOFT_SHAREPOINT_SCOPE = []string{
		"Sites.Read",
		"Sites.Read.All",
		"Files.ReadWrite",
		"offline_access",
		"Sites.ReadWrite.All",
		"Files.Read",
		"Files.Read.All",
		"Files.ReadWrite",
		"Files.ReadWrite.All",
	}
	MICROSOFT_SHAREPOINT_CONNECT = "/connect-knowledge/share-point"
)

func NewMicrosoftAuthenticationConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) MicrosoftConnect {
	return MicrosoftConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		microsoftOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), MICROSOFT_AUTHENTICATION_URL),
			ClientID:     cfg.MicrosoftClientId,
			ClientSecret: cfg.MicrosoftClientSecret,
			Scopes:       MICROSOFT_AUTHENTICATION_SCOPE,
			Endpoint:     microsoft.AzureADEndpoint("common"),
		},
		logger: logger,
	}
}

func NewMicrosoftSharepointConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) MicrosoftConnect {
	return MicrosoftConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		microsoftOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), MICROSOFT_SHAREPOINT_CONNECT),
			ClientID:     cfg.MicrosoftClientId,
			ClientSecret: cfg.MicrosoftClientSecret,
			Scopes:       MICROSOFT_SHAREPOINT_SCOPE,
			Endpoint:     microsoft.AzureADEndpoint("common"),
		},
		logger: logger,
	}
}

func NewMicrosoftOnedriveConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) MicrosoftConnect {
	return MicrosoftConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		microsoftOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), MICROSOFT_ONEDRIVE_CONNECT),
			ClientID:     cfg.MicrosoftClientId,
			ClientSecret: cfg.MicrosoftClientSecret,
			Scopes:       MICROSOFT_ONEDRIVE_SCOPE,
			Endpoint:     microsoft.LiveConnectEndpoint,
		},
		logger: logger,
	}
}

func (microsoft *MicrosoftConnect) codeVerifier(verifier string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(verifier))
}

func (microsoft *MicrosoftConnect) codeChallenge(verifier string) string {
	hash := sha256.New()
	hash.Write([]byte(verifier))
	sha := hash.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sha)
}

func (microsoft *MicrosoftConnect) AuthCodeURL(state string) string {
	codeChallenge := microsoft.codeChallenge(microsoft.codeVerifier(state))
	return microsoft.microsoftOauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("code_challenge", codeChallenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

type MicrosoftTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	Expiry       int64  `json:"expires_in"`
}

func (microsoft *MicrosoftConnect) Token(c context.Context, code string, state string) (*oauth2.Token, error) {
	microsoft.log.Debugf("requesting to get token from microsoft %v", code)
	client := resty.New()

	// Build the request body
	data := url.Values{
		"client_id":     {microsoft.microsoftOauthConfig.ClientID},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {microsoft.microsoftOauthConfig.RedirectURL},
		"code_verifier": {microsoft.codeVerifier(state)},
	}

	// Send the POST request
	resp, err := client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody(data.Encode()).
		Post(microsoft.microsoftOauthConfig.Endpoint.TokenURL)

	if err != nil {
		log.Printf("Error while creating request: %v", err)
		return nil, err
	}

	if resp.IsError() {
		log.Printf("Error response: %s", resp.String())
		return nil, fmt.Errorf("failed to fetch token: %s", resp.Status())
	}

	var tokenResponse MicrosoftTokenResponse
	err = json.Unmarshal(resp.Body(), &tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token response: %v", err)
	}

	expiryDuration := time.Duration(tokenResponse.Expiry) * time.Second

	token := &oauth2.Token{
		AccessToken:  tokenResponse.AccessToken,
		TokenType:    tokenResponse.TokenType,
		RefreshToken: tokenResponse.RefreshToken,
		Expiry:       time.Now().Add(expiryDuration),
	}

	return token, nil
}

type Folder struct {
	ChildCount int `json:"childCount"`
}

type File struct {
	MimeType string `json:"mimeType"`
}

// Define data structures for parsing API responses
type OneDriveFile struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Parent   string    `json:"parent"`
	Folder   *Folder   `json:"folder,omitempty"`
	File     *File     `json:"file,omitempty"`
	Created  time.Time `json:"createdDateTime"`
	Modified time.Time `json:"lastModifiedDateTime"`
	MimeType string    `json:"mimeType"`
}

type OneDriveFiles struct {
	Value []*OneDriveFile `json:"value"`
}

func (microsft *MicrosoftConnect) OneDriveFiles(ctx context.Context,
	token *oauth2.Token,
	q *string,
	pageToken *string) (*OneDriveFiles, error) {
	client := microsft.microsoftOauthConfig.Client(ctx, token)
	restyClient := resty.NewWithClient(client)
	// Example: Fetch files in the root folder of OneDrive
	rootFolderID := "root" // "root" for the root folder, or a specific folder ID
	val, err := microsft.onedriveFetchAllFilesAndFolders(restyClient, rootFolderID, rootFolderID)
	if err != nil {
		microsft.log.Errorf("failed to get all the files and folder in given one drive")
		return nil, err
	}
	return &OneDriveFiles{Value: val}, nil
}

// One drive implimentation
func (microsft *MicrosoftConnect) onedriveFetchAllFilesAndFolders(client *resty.Client, folderID, parent string) ([]*OneDriveFile, error) {
	var allFiles []*OneDriveFile

	files, err := microsft.onedriveFetchFilesInFolder(client, folderID)
	if err != nil {
		microsft.log.Errorf("failed to fetch files in folder %v", err)
		return nil, err
	}

	for _, file := range files.Value {
		file.Parent = parent
		if file.Folder != nil {
			file.MimeType = "application/folder"
			allFiles = append(allFiles, file)

			// this is folder
			// If the item is a folder, recursively fetch its contents
			subFiles, err := microsft.onedriveFetchAllFilesAndFolders(client, file.ID, fmt.Sprintf("%s/%s", parent, file.Name))
			if err != nil {
				microsft.log.Errorf("failed to get recursively the files in subfolder %v", err)
				return nil, err
			}
			allFiles = append(allFiles, subFiles...)
		}

		if file.File != nil {
			file.MimeType = file.File.MimeType
			allFiles = append(allFiles, file)
		}
	}

	return allFiles, nil
}

func (microsft *MicrosoftConnect) onedriveFetchFilesInFolder(client *resty.Client, folderID string) (*OneDriveFiles, error) {
	var files OneDriveFiles

	// OneDrive API URL for retrieving files in a specific folder
	apiURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s/children", folderID)

	_, err := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetResult(&files).
		Get(apiURL)

	if err != nil {
		microsft.log.Errorf("api call to one drive failed, err = %v", err)
		return nil, fmt.Errorf("error fetching files: %v", err)
	}
	return &files, nil
}

// Define data structures for parsing API responses
type SharePointFile struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Folder   *Folder   `json:"folder,omitempty"`
	File     *File     `json:"file,omitempty"`
	Created  time.Time `json:"createdDateTime"`
	Modified time.Time `json:"lastModifiedDateTime"`
}

type SharePointFiles struct {
	Value    []*SharePointFile `json:"value"`
	NextLink string            `json:"@odata.nextLink"`
	Count    int               `json:"@odata.count"`
}

type SharePointSite struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (microsft *MicrosoftConnect) SharePointFiles(ctx context.Context,
	token *oauth2.Token,
	q *string,
	pageToken *string) (*SharePointFiles, error) {
	client := microsft.microsoftOauthConfig.Client(ctx, token)
	restyClient := resty.NewWithClient(client)

	// Fetch the default site ID dynamically
	siteID, err := microsft.sharepointFetchDefaultSiteID(restyClient)
	if err != nil {
		log.Fatalf("Error fetching site ID: %v", err)
	}

	// Example: Fetch all files and folders in SharePoint
	rootFolderID := "root" // "root" for the root folder, or a specific folder ID
	return microsft.sharepointFetchAllFilesAndFolders(restyClient, siteID, rootFolderID)
}

func (microsft *MicrosoftConnect) sharepointFetchDefaultSiteID(client *resty.Client) (string, error) {
	var site SharePointSite

	// SharePoint API URL for retrieving the default site ID
	apiURL := "https://graph.microsoft.com/v1.0/sites/root"

	_, err := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetResult(&site).
		Get(apiURL)

	if err != nil {
		return "", fmt.Errorf("error fetching default site ID: %v", err)
	}

	return site.ID, nil
}

func (microsft *MicrosoftConnect) sharepointFetchAllFilesAndFolders(client *resty.Client, siteID, folderID string) (*SharePointFiles, error) {
	// var allFiles []*SharePointFile

	return microsft.sharepointFetchAllFilesAndFolders(client, siteID, folderID)
	// if err != nil {
	// 	return nil, err
	// }

	// for _, file := range files.Value {
	// 	allFiles = append(allFiles, file)
	// 	if file.Folder != nil {
	// 		// If the item is a folder, recursively fetch its contents
	// 		subFiles, err := microsft.sharepointFetchAllFilesAndFolders(client, siteID, file.ID)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		allFiles = append(allFiles, subFiles...)
	// 	}
	// }

	// return allFiles, nil
}

func (microsft *MicrosoftConnect) sharepointFetchFilesInFolder(client *resty.Client, siteID, folderID string) (*SharePointFiles, error) {
	var files SharePointFiles

	// SharePoint API URL for retrieving files in a specific folder
	apiURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/sites/%s/drive/items/%s/children", siteID, folderID)

	_, err := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetResult(&files).
		Get(apiURL)

	if err != nil {
		return nil, fmt.Errorf("error fetching files: %v", err)
	}

	return &files, nil
}
