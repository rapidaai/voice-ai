package internal_connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	config "github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
)

type GoogleConnect struct {
	ExternalConnect
	logger            commons.Logger
	googleOauthConfig oauth2.Config
}

var (
	GOOGLE_AUTHENTICATION_STATE = "google"
	GOOGLE_AUTHENTICATION_SCOPE = []string{"email", "profile"}
	GOOGLE_AUTHENTICATION_URL   = "/auth/signin"

	// GOOGLE_DRIVE_STATE       = "connect/"
	GOOGLE_DRIVE_SCOPE = []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/drive.readonly",
		"https://www.googleapis.com/auth/drive.file",
		"https://www.googleapis.com/auth/drive.metadata.readonly",
	}
	GOOGLE_DRIVE_CONNECT_URL = "/connect-knowledge/google-drive"

	GOOGLE_GMAIL_SCOPE = []string{
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/gmail.compose",
		"https://www.googleapis.com/auth/gmail.send",
	}
	GOOGLE_GMAIL_CONNECT_URL = "/connect-action/gmail"
)

func NewGoogleAuthenticationConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) GoogleConnect {
	return GoogleConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		googleOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GOOGLE_AUTHENTICATION_URL),
			ClientID:     oauthCfg.GoogleClientId,
			ClientSecret: oauthCfg.GoogleClientSecret,
			Scopes:       GOOGLE_AUTHENTICATION_SCOPE,
			Endpoint:     google.Endpoint,
		},
		logger: logger,
	}
}

func NewGoogleDriveConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) GoogleConnect {
	return GoogleConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		googleOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GOOGLE_DRIVE_CONNECT_URL),
			ClientID:     oauthCfg.GoogleClientId,
			ClientSecret: oauthCfg.GoogleClientSecret,
			Scopes:       GOOGLE_DRIVE_SCOPE,
			Endpoint:     google.Endpoint,
		},
		logger: logger,
	}
}

func NewGmailConnect(cfg *config.WebAppConfig, oauthCfg *config.OAuthConfig, logger commons.Logger, postgres connectors.PostgresConnector) GoogleConnect {
	return GoogleConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		googleOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GOOGLE_GMAIL_CONNECT_URL),
			ClientID:     oauthCfg.GoogleClientId,
			ClientSecret: oauthCfg.GoogleClientSecret,
			Scopes:       GOOGLE_GMAIL_SCOPE,
			Endpoint:     google.Endpoint,
		},
		logger: logger,
	}
}

/*
*

Google implimentation
*/

type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

func (gtr *GoogleTokenResponse) Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  gtr.AccessToken,
		TokenType:    gtr.TokenType,
		RefreshToken: gtr.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(gtr.ExpiresIn) * time.Second),
	}
}

func (gtr *GoogleTokenResponse) Map() map[string]interface{} {
	return map[string]interface{}{
		"accessToken":  gtr.AccessToken,
		"tokenType":    gtr.TokenType,
		"refreshToken": gtr.RefreshToken,
		"expiry":       time.Now().Add(time.Duration(gtr.ExpiresIn) * time.Second),
		"scope":        gtr.Scope,
	}
}

// "google"
func (wAuthApi *GoogleConnect) AuthCodeURL(state string) string {
	return wAuthApi.googleOauthConfig.AuthCodeURL(state)
}

func (wAuthApi *GoogleConnect) Token(c context.Context, code string) (ExternalConnectToken, error) {
	tokenEndpoint := wAuthApi.googleOauthConfig.Endpoint.TokenURL

	// Prepare the data for the request
	data := map[string]string{
		"code":          code,
		"client_id":     wAuthApi.googleOauthConfig.ClientID,
		"client_secret": wAuthApi.googleOauthConfig.ClientSecret,
		"redirect_uri":  wAuthApi.googleOauthConfig.RedirectURL,
		"grant_type":    "authorization_code",
	}

	// Make the request
	resp, err := wAuthApi.NewHttpClient().R().
		SetContext(c).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(data).
		Post(tokenEndpoint)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, errors.New("failed to exchange token")
	}

	// Parse the response
	var tokenResponse GoogleTokenResponse
	if err := json.Unmarshal(resp.Body(), &tokenResponse); err != nil {
		return nil, err
	}
	return &tokenResponse, nil
}
func (wAuthApi *GoogleConnect) GoogleUserInfo(c context.Context, code string) (*OpenID, error) {
	externalToken, err := wAuthApi.Token(c, code)
	if err != nil {
		wAuthApi.logger.Errorf("google authentication exchange failed %v", err)
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}
	token := externalToken.Token()
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		wAuthApi.logger.Errorf("unable to get userinfo using the access token %v", err)
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()

	var content OpenID
	err = json.NewDecoder(response.Body).Decode(&content)
	content.Source = "google"
	content.Token = token.AccessToken
	if err != nil {
		wAuthApi.logger.Errorf("unable to decode the response body of the user info %v", err)
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}
	return &content, nil
}

type GoogleFile struct {
	Id       string `json:"id"`
	Title    string `json:"title"`
	Folder   string `json:"folder"`
	FileSize int64  `json:"fileSize,omitempty,string"`
	MimeType string `json:"mimeType"`
}
type GoogleDriveFiles struct {
	Items         []*GoogleFile `json:"items,omitempty"`
	NextPageToken string        `json:"nextPageToken,omitempty"`
}

// "google"
func (wAuthApi *GoogleConnect) GoogleDriveFiles(ctx context.Context,
	token *oauth2.Token,
	q *string,
	pageToken *string) (*GoogleDriveFiles, error) {
	driveClient := wAuthApi.googleOauthConfig.Client(ctx, token)
	googleDrive, err := drive.NewService(ctx, option.WithHTTPClient(driveClient))
	if err != nil {
		wAuthApi.logger.Errorf("google connect with drive files %+v", err)
		return nil, err
	}

	driveRequest := googleDrive.Files.List()
	if pageToken != nil {
		driveRequest = driveRequest.PageToken(*pageToken)
	}
	if q != nil {
		// Q("mimeType='application/vnd.google-apps.folder'").
		driveRequest = driveRequest.Q(*q)
	}
	driveRequest.MaxResults(20)
	driveRequest = driveRequest.
		Fields("nextPageToken", "items(id,fileSize,mimeType,modifiedDate,originalFilename,title,downloadUrl,fileExtension,parents)")

	fls, err := driveRequest.Do()
	if err != nil {
		wAuthApi.logger.Errorf("google connect with drive files %+v", err)
		return nil, err
	}

	var gdf GoogleDriveFiles
	for _, fl := range fls.Items {
		gdf.Items = append(gdf.Items, &GoogleFile{
			Id:       fl.Id,
			Title:    fl.Title,
			Folder:   wAuthApi.GetFolderName(googleDrive, fl.Parents),
			FileSize: fl.FileSize,
			MimeType: fl.MimeType,
		})
	}
	gdf.NextPageToken = fls.NextPageToken
	return &gdf, nil
}

func (wAuthApi *GoogleConnect) GetFolderName(srv *drive.Service, parents []*drive.ParentReference) string {
	if len(parents) == 0 {
		return "Root"
	}
	parentID := parents[0]
	parent, err := srv.Files.Get(parentID.Id).Fields("title").Do()
	if err != nil {
		return "Unknown"
	}
	return parent.Title
}
