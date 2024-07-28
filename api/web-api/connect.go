package web_api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	config "github.com/lexatic/web-backend/config"
	internal_connects "github.com/lexatic/web-backend/internal/connects"
	internal_services "github.com/lexatic/web-backend/internal/services"
	internal_vault_service "github.com/lexatic/web-backend/internal/services/vault"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	gorm_types "github.com/lexatic/web-backend/pkg/models/gorm/types"
	"github.com/lexatic/web-backend/pkg/types"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"golang.org/x/oauth2"
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

	// jiraConnect
	jiraConnect  internal_connects.AtlassianConnect
	slackConnect internal_connects.SlackConnect
	gmailConnect internal_connects.GoogleConnect

	//
	vaultService internal_services.VaultService
}

type webConnectRPCApi struct {
	webConnectApi
}

type webConnectGRPCApi struct {
	webConnectApi
}

const (
	KN_GOOGLE_DRIVE string = "knowledge/google/google-drive"
	KN_NOTION       string = "knowledge/notion"
	KN_CONFLUENCE   string = "knowledge/atlassian/confluence"

	KN_SHARE_POINT string = "knowledge/microsoft/share-point"
	KN_ONE_DRIVE   string = "knowledge/microsoft/one-drive"
	KN_GITHUB_CODE string = "knowledge/github/github-code"
	KN_GITLAB_CODE string = "knowledge/gitlab/gitlab-code"

	// action

	AN_GOOGLE_DRIVE string = "action/google/google-drive"
	AN_GOOGLE_GMAIL string = "action/google/gmail"
	AN_JIRA         string = "action/atlassian/jira"
	AN_SLACK        string = "action/slack"
	AN_TWILIO       string = "action/twilio"
)

// KnowledgeConnect implements lexatic_backend.ConnectServiceServer.
func (wConnectApi *webConnectGRPCApi) KnowledgeConnect(ctx context.Context, kcr *web_api.KnowledgeConnectRequest) (*web_api.KnowledgeConnectResponse, error) {
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		wConnectApi.logger.Errorf("unauthenticated request to fork endpoint")
		return utils.AuthenticateError[web_api.KnowledgeConnectResponse]()
	}
	decodedState, err := wConnectApi.googleDriveConnect.DecodeState(ctx, auth, kcr.State)
	if err != nil {
		wConnectApi.logger.Errorf("illegal state for oauth %v", err)
		return utils.Error[web_api.KnowledgeConnectResponse](err, "illegal state for oauth")
	}

	var tokenInfo *oauth2.Token
	switch decodedState.ToolConnect {
	case KN_GOOGLE_DRIVE:
		tokenInfo, err = wConnectApi.googleDriveConnect.Token(ctx, kcr.Code)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.KnowledgeConnectResponse](err, "illegal state for getting oauth2 token ")
		}

	case KN_NOTION:
		tokenInfo, err = wConnectApi.notionConnect.Token(ctx, kcr.Code)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.KnowledgeConnectResponse](err, "illegal state for getting oauth2 token ")
		}

	case KN_CONFLUENCE:
		tokenInfo, err = wConnectApi.confluenceConnect.Token(ctx, kcr.Code)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.KnowledgeConnectResponse](err, "illegal state for getting oauth2 token ")
		}

	case KN_SHARE_POINT:
		tokenInfo, err = wConnectApi.microsoftSharepointConnect.Token(ctx, kcr.Code, kcr.State)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.KnowledgeConnectResponse](err, "illegal state for getting oauth2 token ")
		}
	case KN_ONE_DRIVE:
		tokenInfo, err = wConnectApi.microsoftOnedriveConnect.Token(ctx, kcr.Code, kcr.State)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.KnowledgeConnectResponse](err, "illegal state for getting oauth2 token ")
		}
	case KN_GITHUB_CODE:
		tokenInfo, err = wConnectApi.githubCodeConnect.Token(ctx, kcr.Code)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.KnowledgeConnectResponse](err, "illegal state for getting oauth2 token ")
		}
	case KN_GITLAB_CODE:
		tokenInfo, err = wConnectApi.gitlabCodeConnect.Token(ctx, kcr.Code)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.KnowledgeConnectResponse](err, "illegal state for getting oauth2 token ")
		}
	default:
		return utils.Error[web_api.KnowledgeConnectResponse](err, "Unknown connector request.")

	}

	credential := map[string]interface{}{
		"accessToken":  tokenInfo.AccessToken,
		"tokenType":    tokenInfo.TokenType,
		"refreshToken": tokenInfo.RefreshToken,
		"expiry":       tokenInfo.Expiry,
		"scope":        kcr.Scope,
		"code":         kcr.Code,
		"connect":      decodedState.ToolConnect,
		"state":        kcr.State,
	}
	if decodedState.Linker == gorm_types.VAULT_LEVEL_ORGANIZATION {
		_, err := wConnectApi.vaultService.CreateOrganizationToolCredential(
			ctx, auth, decodedState.ToolId, "connected-org-tool", credential)
		if err != nil {
			return utils.Error[web_api.KnowledgeConnectResponse](err, "Unable to store the generated token")
		}
	}
	if decodedState.Linker == gorm_types.VAULT_LEVEL_USER {
		_, err := wConnectApi.vaultService.CreateUserToolCredential(
			ctx, auth, decodedState.ToolId, "connected-user-tool", credential)
		if err != nil {
			return utils.Error[web_api.KnowledgeConnectResponse](err, "Unable to store the generated token")
		}

	}
	// decodedState.Linker
	return &web_api.KnowledgeConnectResponse{
		Success:    true,
		Code:       200,
		ToolId:     decodedState.ToolId,
		RedirectTo: decodedState.RedirectTo,
	}, nil
}

// ActionConnect implements lexatic_backend.ConnectServiceServer.
func (wConnectApi *webConnectGRPCApi) ActionConnect(ctx context.Context, acr *web_api.ActionConnectRequest) (*web_api.ActionConnectResponse, error) {
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		wConnectApi.logger.Errorf("unauthenticated request to fork endpoint")
		return utils.AuthenticateError[web_api.ActionConnectResponse]()
	}
	decodedState, err := wConnectApi.googleDriveConnect.DecodeState(ctx, auth, acr.State)
	if err != nil {
		wConnectApi.logger.Errorf("illegal state for oauth %v", err)
		return utils.Error[web_api.ActionConnectResponse](err, "illegal state for oauth")
	}

	var tokenInfo *oauth2.Token
	switch decodedState.ToolConnect {
	case AN_GOOGLE_DRIVE:
		tokenInfo, err = wConnectApi.googleDriveConnect.Token(ctx, acr.Code)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.ActionConnectResponse](err, "illegal state for getting oauth2 token ")
		}

	case AN_GOOGLE_GMAIL:
		tokenInfo, err = wConnectApi.gmailConnect.Token(ctx, acr.Code)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.ActionConnectResponse](err, "illegal state for getting oauth2 token ")
		}

	case AN_SLACK:
		tokenInfo, err = wConnectApi.gmailConnect.Token(ctx, acr.Code)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.ActionConnectResponse](err, "illegal state for getting oauth2 token ")
		}

	case AN_JIRA:
		tokenInfo, err = wConnectApi.jiraConnect.Token(ctx, acr.Code)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[web_api.ActionConnectResponse](err, "illegal state for getting oauth2 token ")
		}
	default:
		return utils.Error[web_api.ActionConnectResponse](err, "Unknown connector request.")

	}

	credential := map[string]interface{}{
		"accessToken":  tokenInfo.AccessToken,
		"tokenType":    tokenInfo.TokenType,
		"refreshToken": tokenInfo.RefreshToken,
		"expiry":       tokenInfo.Expiry,
		"scope":        acr.Scope,
		"code":         acr.Code,
		"connect":      decodedState.ToolConnect,
		"state":        acr.State,
	}
	if decodedState.Linker == gorm_types.VAULT_LEVEL_ORGANIZATION {
		_, err := wConnectApi.vaultService.CreateOrganizationToolCredential(
			ctx, auth, decodedState.ToolId, "connected-org-tool", credential)
		if err != nil {
			return utils.Error[web_api.ActionConnectResponse](err, "Unable to store the generated token")
		}
	}
	if decodedState.Linker == gorm_types.VAULT_LEVEL_USER {
		_, err := wConnectApi.vaultService.CreateUserToolCredential(
			ctx, auth, decodedState.ToolId, "connected-user-tool", credential)
		if err != nil {
			return utils.Error[web_api.ActionConnectResponse](err, "Unable to store the generated token")
		}

	}
	// decodedState.Linker
	return &web_api.ActionConnectResponse{
		Success:    true,
		Code:       200,
		ToolId:     decodedState.ToolId,
		RedirectTo: decodedState.RedirectTo,
	}, nil
}

func NewConnectRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) *webConnectRPCApi {
	return &webConnectRPCApi{
		webConnectApi{
			cfg:                config,
			logger:             logger,
			postgres:           postgres,
			githubCodeConnect:  internal_connects.NewGithubCodeConnect(config, logger, postgres),
			gitlabCodeConnect:  internal_connects.NewGitlabCodeConnect(config, logger, postgres),
			googleDriveConnect: internal_connects.NewGoogleDriveConnect(config, logger, postgres),
			confluenceConnect:  internal_connects.NewConfluenceConnect(config, logger, postgres),
			notionConnect:      internal_connects.NewNotionWorkplaceConnect(config, logger, postgres),

			//
			microsoftSharepointConnect: internal_connects.NewMicrosoftSharepointConnect(config, logger, postgres),
			microsoftOnedriveConnect:   internal_connects.NewMicrosoftOnedriveConnect(config, logger, postgres),
			//
			vaultService: internal_vault_service.NewVaultService(logger, postgres),

			slackConnect: internal_connects.NewSlackActionConnect(config, logger, postgres),
			jiraConnect:  internal_connects.NewJiraConnect(config, logger, postgres),
			gmailConnect: internal_connects.NewGmailConnect(config, logger, postgres),
		},
	}
}

func NewConnectGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) web_api.ConnectServiceServer {
	return &webConnectGRPCApi{
		webConnectApi{
			cfg:                config,
			logger:             logger,
			postgres:           postgres,
			githubCodeConnect:  internal_connects.NewGithubCodeConnect(config, logger, postgres),
			gitlabCodeConnect:  internal_connects.NewGitlabCodeConnect(config, logger, postgres),
			googleDriveConnect: internal_connects.NewGoogleDriveConnect(config, logger, postgres),
			confluenceConnect:  internal_connects.NewConfluenceConnect(config, logger, postgres),
			notionConnect:      internal_connects.NewNotionWorkplaceConnect(config, logger, postgres),

			//
			microsoftSharepointConnect: internal_connects.NewMicrosoftSharepointConnect(config, logger, postgres),
			microsoftOnedriveConnect:   internal_connects.NewMicrosoftOnedriveConnect(config, logger, postgres),
			//
			vaultService: internal_vault_service.NewVaultService(logger, postgres),

			slackConnect: internal_connects.NewSlackActionConnect(config, logger, postgres),
			jiraConnect:  internal_connects.NewJiraConnect(config, logger, postgres),
			gmailConnect: internal_connects.NewGmailConnect(config, logger, postgres),
		},
	}
}

func (connectApi *webConnectRPCApi) buildConnectParameter(c *gin.Context, idx string) (string, error) {

	link, ok := c.GetQuery("link")
	if !ok {
		connectApi.logger.Errorf("google drive connect, the link is illegal.")
		return "", errors.New("google drive connect, the link is illegal")
	}

	// redirection after successful connect
	redirectTo, ok := c.GetQuery("redirect_to")
	if !ok {
		connectApi.logger.Errorf("google drive connect, there isn't any redirect url.")
		return "", errors.New("google drive connect, there isn't any redirect url")
	}

	linkId, ok := c.GetQuery("link_id")
	if !ok {
		connectApi.logger.Errorf("google drive connect, there isn't any link id to configure the link.")
		return "", errors.New("google drive connect, there isn't any link_id")
	}

	toolId, ok := c.GetQuery("tool_id")
	if !ok {
		connectApi.logger.Errorf("google drive connect, there isn't any link id to configure the link.")
		return "", errors.New("google drive connect, there isn't any tool_id")
	}

	linkerId, err := strconv.ParseUint(linkId, 10, 64)
	if err != nil {
		connectApi.logger.Errorf("google drive connect, the link id is not uint 64")
		return "", err
	}

	toolProviderId, err := strconv.ParseUint(toolId, 10, 64)
	if err != nil {
		connectApi.logger.Errorf("google drive connect, the tool id is not uint 64")
		return "", err
	}
	// connector string,
	// linker Linker, linkerId uint64, redirect string
	state, err := connectApi.notionConnect.EncodeState(
		c,
		toolProviderId,
		string(idx),
		gorm_types.VaultLevel(link),
		linkerId,
		redirectTo)

	if err != nil {
		connectApi.logger.Errorf("unauthenticated request to notion connect")
		return "", err
	}
	return state, nil

}
func (connectApi *webConnectRPCApi) ConfluenceConnect(c *gin.Context) {

	state, err := connectApi.buildConnectParameter(c, KN_CONFLUENCE)
	if err != nil {
		return
	}
	url := connectApi.confluenceConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GoogleDriveConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, KN_GOOGLE_DRIVE)
	if err != nil {
		return
	}
	url := connectApi.googleDriveConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GithubCodeConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, KN_GITHUB_CODE)
	if err != nil {
		return
	}
	url := connectApi.githubCodeConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for github connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GitlabCodeConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, KN_GITLAB_CODE)
	if err != nil {
		return
	}
	url := connectApi.gitlabCodeConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for gitlab connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) MicrosoftSharepointConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, KN_SHARE_POINT)
	if err != nil {
		return
	}
	url := connectApi.microsoftSharepointConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for sharepoint connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) MicrosoftOnedriveConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, KN_ONE_DRIVE)
	if err != nil {
		return
	}
	url := connectApi.microsoftOnedriveConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for microsoft onedrive connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) NotionConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, KN_NOTION)
	if err != nil {
		return
	}
	url := connectApi.notionConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for notion connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

// all the action connect
func (connectApi *webConnectRPCApi) JiraActionConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, AN_JIRA)
	if err != nil {
		return
	}
	url := connectApi.jiraConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GmailActionConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, AN_GOOGLE_GMAIL)
	if err != nil {
		return
	}
	url := connectApi.gmailConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) SlackActionConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, AN_SLACK)
	if err != nil {
		return
	}
	url := connectApi.slackConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectGRPCApi) GetConnectorFiles(ctx context.Context,
	r *web_api.GetConnectorFilesRequest) (*web_api.GetConnectorFilesResponse, error) {
	//
	// toolId uint64, q *string, pageToken *string
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		connectApi.logger.Errorf("unauthenticated request to fork endpoint")
		return utils.AuthenticateError[web_api.GetConnectorFilesResponse]()
	}
	crd, err := connectApi.vaultService.GetToolCredential(
		ctx, auth, r.GetToolId())
	if err != nil {
		connectApi.logger.Errorf("unable to get tool credentials %v", err)
		return utils.Error[web_api.GetConnectorFilesResponse](err, "Unable to get tool credential to get list of files.")
	}
	token, connect, err := connectApi.googleDriveConnect.ToToken(crd.Value)
	if err != nil {
		connectApi.logger.Errorf("unable to get tool credentials %v", err)
		return utils.Error[web_api.GetConnectorFilesResponse](err, "Unable to get tool credential to get list of files.")
	}

	repeatArgs := make(map[string]string)
	var q, pageToken *string
	for _, x := range r.GetCriterias() {
		if x.GetKey() == "query" {
			q = &x.Value
			repeatArgs["query"] = x.Value
		}
		if x.GetKey() == "page_token" {
			pageToken = &x.Value

		}
	}

	switch connect {
	case KN_GOOGLE_DRIVE:
		fls, err := connectApi.googleDriveConnect.GoogleDriveFiles(ctx, token, q, pageToken)
		if err != nil {
			connectApi.logger.Errorf("unable to get tool credentials %v", err)
			return utils.Error[web_api.GetConnectorFilesResponse](err, "Unable to get files ")
		}

		return utils.Success[web_api.GetConnectorFilesResponse](fls.Items)
	case KN_CONFLUENCE:
		fls, err := connectApi.confluenceConnect.ConfluencePages(ctx, token, q, pageToken)
		if err != nil {
			connectApi.logger.Errorf("unable to get tool credentials %v", err)
			return utils.Error[web_api.GetConnectorFilesResponse](err, "Unable to get files ")
		}
		return utils.Success[web_api.GetConnectorFilesResponse](fls.Items)
	case KN_NOTION:
		fls, err := connectApi.notionConnect.NotionPages(ctx, token, q, pageToken)
		if err != nil {
			connectApi.logger.Errorf("unable to get tool credentials %v", err)
			return utils.Error[web_api.GetConnectorFilesResponse](err, "Unable to get files ")
		}
		return utils.Success[web_api.GetConnectorFilesResponse](fls.Results)
	case KN_ONE_DRIVE:
		fls, err := connectApi.microsoftOnedriveConnect.OneDriveFiles(ctx, token, q, pageToken)
		if err != nil {
			connectApi.logger.Errorf("unable to get tool credentials %v", err)
			return utils.Error[web_api.GetConnectorFilesResponse](err, "Unable to get files ")
		}
		return utils.Success[web_api.GetConnectorFilesResponse](fls.Value)
	case KN_SHARE_POINT:
		fls, err := connectApi.microsoftSharepointConnect.SharePointFiles(ctx, token, q, pageToken)
		if err != nil {
			connectApi.logger.Errorf("unable to get tool credentials %v", err)
			return utils.Error[web_api.GetConnectorFilesResponse](err, "Unable to get files ")
		}
		return utils.Success[web_api.GetConnectorFilesResponse](fls.Value)
	default:
		return utils.AuthenticateError[web_api.GetConnectorFilesResponse]()
	}
}
