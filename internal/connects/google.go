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

func NewGoogleAuthenticationConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) GoogleConnect {
	return GoogleConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		googleOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GOOGLE_AUTHENTICATION_URL),
			ClientID:     cfg.GoogleClientId,
			ClientSecret: cfg.GoogleClientSecret,
			Scopes:       GOOGLE_AUTHENTICATION_SCOPE,
			Endpoint:     google.Endpoint,
		},
		logger: logger,
	}
}

func NewGoogleDriveConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) GoogleConnect {
	return GoogleConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		googleOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GOOGLE_DRIVE_CONNECT_URL),
			ClientID:     cfg.GoogleClientId,
			ClientSecret: cfg.GoogleClientSecret,
			Scopes:       GOOGLE_DRIVE_SCOPE,
			Endpoint:     google.Endpoint,
		},
		logger: logger,
	}
}

func NewGmailConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) GoogleConnect {
	return GoogleConnect{
		ExternalConnect: NewExternalConnect(cfg, logger, postgres),
		googleOauthConfig: oauth2.Config{
			RedirectURL:  fmt.Sprintf("%s%s", cfg.BaseUrl(), GOOGLE_GMAIL_CONNECT_URL),
			ClientID:     cfg.GoogleClientId,
			ClientSecret: cfg.GoogleClientSecret,
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

// "google"
func (wAuthApi *GoogleConnect) AuthCodeURL(state string) string {
	return wAuthApi.googleOauthConfig.AuthCodeURL(state)
}

func (wAuthApi *GoogleConnect) Token(c context.Context, code string) (*oauth2.Token, error) {
	return wAuthApi.googleOauthConfig.Exchange(c, code)
}
func (wAuthApi *GoogleConnect) GoogleUserInfo(c context.Context, code string) (*OpenID, error) {
	token, err := wAuthApi.Token(c, code)
	if err != nil {
		wAuthApi.logger.Errorf("google authentication exchange failed %v", err)
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}
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
	driveRequest = driveRequest.
		Fields("nextPageToken", "items(id,fileSize,mimeType,modifiedDate,originalFilename,title,downloadUrl,fileExtension,parents)")

	fls, err := driveRequest.Do()
	if err != nil {
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
