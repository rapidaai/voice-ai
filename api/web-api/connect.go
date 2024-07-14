package web_api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	config "github.com/lexatic/web-backend/config"
	internal_connects "github.com/lexatic/web-backend/internal/connects"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type webConnectApi struct {
	cfg      *config.AppConfig
	logger   commons.Logger
	postgres connectors.PostgresConnector
	// code
	githubCodeConnect internal_connects.GithubConnect
	gitlabCodeConnect internal_connects.GitlabConnect

	// google workspace
	googleDriveConnect internal_connects.GoogleConnect

	// microsft
	microsoftSharepointConnect internal_connects.MicrosoftConnect
	microsoftOnedriveConnect   internal_connects.MicrosoftConnect

	// notion
	notionConnect internal_connects.NotionConnect

	// confluence
	confluenceConnect internal_connects.AtlassianConnect
}

type webConnectRPCApi struct {
	webConnectApi
}

type webConnectGRPCApi struct {
	webConnectApi
}

// KnowledgeConnect implements lexatic_backend.ConnectServiceServer.
func (*webConnectGRPCApi) KnowledgeConnect(context.Context, *web_api.KnowledgeConnectRequest) (*web_api.KnowledgeConnecctResponse, error) {
	panic("unimplemented")
}

// ToolConnect implements lexatic_backend.ConnectServiceServer.
func (*webConnectGRPCApi) ToolConnect(context.Context, *web_api.ToolConnectRequest) (*web_api.ToolConnetResponse, error) {
	panic("unimplemented")
}

func NewConnectRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) *webConnectRPCApi {
	return &webConnectRPCApi{
		webConnectApi{
			cfg:                config,
			logger:             logger,
			postgres:           postgres,
			githubCodeConnect:  internal_connects.NewGithubCodeConnect(config, logger),
			gitlabCodeConnect:  internal_connects.NewGitlabCodeConnect(config, logger),
			googleDriveConnect: internal_connects.NewGoogleDriveConnect(config, logger),
			confluenceConnect:  internal_connects.NewConfluenceConnect(config, logger),
			notionConnect:      internal_connects.NewNotionWorkplaceConnect(config, logger),

			//
			microsoftSharepointConnect: internal_connects.NewMicrosoftSharepointConnect(config, logger),
			microsoftOnedriveConnect:   internal_connects.NewMicrosoftOnedriveConnect(config, logger),
		},
	}
}

func NewConnectGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) web_api.ConnectServiceServer {
	return &webConnectGRPCApi{
		webConnectApi{
			cfg:                config,
			logger:             logger,
			postgres:           postgres,
			githubCodeConnect:  internal_connects.NewGithubCodeConnect(config, logger),
			gitlabCodeConnect:  internal_connects.NewGitlabCodeConnect(config, logger),
			googleDriveConnect: internal_connects.NewGoogleDriveConnect(config, logger),
			confluenceConnect:  internal_connects.NewConfluenceConnect(config, logger),
			notionConnect:      internal_connects.NewNotionWorkplaceConnect(config, logger),

			//
			microsoftSharepointConnect: internal_connects.NewMicrosoftSharepointConnect(config, logger),
			microsoftOnedriveConnect:   internal_connects.NewMicrosoftOnedriveConnect(config, logger),
		},
	}
}

func (connectApi *webConnectRPCApi) ConfluenceConnect(c *gin.Context) {
	state, ok := c.GetQuery("state")
	if !ok {
		state = "connect-application"
	}
	url := connectApi.confluenceConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GoogleDriveConnect(c *gin.Context) {
	state, ok := c.GetQuery("state")
	if !ok {
		state = "connect-application"
	}
	url := connectApi.googleDriveConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GithubCodeConnect(c *gin.Context) {
	state, ok := c.GetQuery("state")
	if !ok {
		state = "connect-application"
	}
	url := connectApi.githubCodeConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for github connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GitlabCodeConnect(c *gin.Context) {
	state, ok := c.GetQuery("state")
	if !ok {
		state = "connect-application"
	}
	url := connectApi.gitlabCodeConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for gitlab connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) MicrosoftSharepointConnect(c *gin.Context) {
	state, ok := c.GetQuery("state")
	if !ok {
		state = "connect-application"
	}
	url := connectApi.microsoftSharepointConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for sharepoint connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) MicrosoftOnedriveConnect(c *gin.Context) {
	state, ok := c.GetQuery("state")
	if !ok {
		state = "connect-application"
	}
	url := connectApi.microsoftOnedriveConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for microsoft onedrive connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectApi) NotionConnect(c *gin.Context) {
	state, ok := c.GetQuery("state")
	if !ok {
		state = "connect-application"
	}
	url := connectApi.notionConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for notion connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}
