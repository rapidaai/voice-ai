package web_api

import (
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	config "github.com/rapidaai/api/web-api/config"
	internal_connects "github.com/rapidaai/api/web-api/internal/connect"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	internal_vault_service "github.com/rapidaai/api/web-api/internal/service/vault"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type webConnectApi struct {
	cfg      *config.WebAppConfig
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

	// hubspot connect
	hubspotConnect internal_connects.HubspotConnect

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

	CRM_HUBSPOT string = "hubspot"
)

func (wConnectApi *webConnectGRPCApi) GeneralConnect(ctx context.Context, kcr *protos.GeneralConnectRequest) (*protos.GeneralConnectResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	userContext, err := iAuth.UserContext()
	if err != nil {
		wConnectApi.logger.Errorf("unable to resolve user identity for connector callback: %v", err)
		return utils.AuthenticateError[protos.GeneralConnectResponse]()
	}
	projectContext, err := iAuth.ProjectContext()
	if err != nil {
		wConnectApi.logger.Errorf("unable to resolve project context for connector callback: %v", err)
		return utils.AuthenticateError[protos.GeneralConnectResponse]()
	}
	decodedState, err := wConnectApi.googleDriveConnect.DecodeState(ctx, iAuth, kcr.State)
	if err != nil {
		wConnectApi.logger.Errorf("illegal state for oauth %v", err)
		return utils.Error[protos.GeneralConnectResponse](err, "illegal state for oauth")
	}

	var tokenInfo internal_connects.ExternalConnectToken
	switch decodedState.Provider {
	case CRM_HUBSPOT:
		tokenInfo, err = wConnectApi.hubspotConnect.Token(ctx, kcr.Code)
		if err != nil {
			wConnectApi.logger.Errorf("illegal while getting token %v", err)
			return utils.Error[protos.GeneralConnectResponse](err, "illegal state for getting oauth2 token ")
		}

	default:
		return utils.Error[protos.GeneralConnectResponse](err, "Unknown connector request.")
	}

	credential := map[string]interface{}{
		"scope":   kcr.GetScope(),
		"code":    kcr.GetCode(),
		"connect": decodedState.Provider,
		"state":   kcr.GetState(),
	}

	for k, v := range tokenInfo.Map() {
		credential[k] = v
	}

	_, err = wConnectApi.vaultService.Create(
		ctx, userContext.UserID, projectContext, decodedState.Provider, "connected-org-tool", credential)
	if err != nil {
		return utils.Error[protos.GeneralConnectResponse](err, "Unable to store the generated token")
	}

	// decodedState.Linker
	return &protos.GeneralConnectResponse{
		Success:    true,
		Code:       200,
		Provider:   decodedState.Provider,
		RedirectTo: decodedState.RedirectTo,
	}, nil
}

func NewConnectRPC(config *config.WebAppConfig, oauthCfg *config.OAuth2Config, logger commons.Logger, postgres connectors.PostgresConnector) *webConnectRPCApi {
	return &webConnectRPCApi{
		webConnectApi{
			cfg:                config,
			logger:             logger,
			postgres:           postgres,
			githubCodeConnect:  internal_connects.NewGithubCodeConnect(config, oauthCfg, logger, postgres),
			gitlabCodeConnect:  internal_connects.NewGitlabCodeConnect(config, oauthCfg, logger, postgres),
			googleDriveConnect: internal_connects.NewGoogleDriveConnect(config, oauthCfg, logger, postgres),
			confluenceConnect:  internal_connects.NewConfluenceConnect(config, oauthCfg, logger, postgres),
			notionConnect:      internal_connects.NewNotionWorkplaceConnect(config, oauthCfg, logger, postgres),

			//
			microsoftSharepointConnect: internal_connects.NewMicrosoftSharepointConnect(config, oauthCfg, logger, postgres),
			microsoftOnedriveConnect:   internal_connects.NewMicrosoftOnedriveConnect(config, oauthCfg, logger, postgres),
			//
			vaultService: internal_vault_service.NewVaultService(logger, postgres),

			slackConnect:   internal_connects.NewSlackActionConnect(config, oauthCfg, logger, postgres),
			jiraConnect:    internal_connects.NewJiraConnect(config, oauthCfg, logger, postgres),
			gmailConnect:   internal_connects.NewGmailConnect(config, oauthCfg, logger, postgres),
			hubspotConnect: internal_connects.NewHubspotConnect(config, oauthCfg, logger, postgres),
		},
	}
}

func NewConnectGRPC(config *config.WebAppConfig,
	oauthCfg *config.OAuth2Config,
	logger commons.Logger, postgres connectors.PostgresConnector) protos.ConnectServiceServer {
	return &webConnectGRPCApi{
		webConnectApi{
			cfg:          config,
			logger:       logger,
			postgres:     postgres,
			vaultService: internal_vault_service.NewVaultService(logger, postgres),

			githubCodeConnect:  internal_connects.NewGithubCodeConnect(config, oauthCfg, logger, postgres),
			gitlabCodeConnect:  internal_connects.NewGitlabCodeConnect(config, oauthCfg, logger, postgres),
			googleDriveConnect: internal_connects.NewGoogleDriveConnect(config, oauthCfg, logger, postgres),
			confluenceConnect:  internal_connects.NewConfluenceConnect(config, oauthCfg, logger, postgres),
			notionConnect:      internal_connects.NewNotionWorkplaceConnect(config, oauthCfg, logger, postgres),

			//
			microsoftSharepointConnect: internal_connects.NewMicrosoftSharepointConnect(config, oauthCfg, logger, postgres),
			microsoftOnedriveConnect:   internal_connects.NewMicrosoftOnedriveConnect(config, oauthCfg, logger, postgres),
			//

			slackConnect:   internal_connects.NewSlackActionConnect(config, oauthCfg, logger, postgres),
			jiraConnect:    internal_connects.NewJiraConnect(config, oauthCfg, logger, postgres),
			gmailConnect:   internal_connects.NewGmailConnect(config, oauthCfg, logger, postgres),
			hubspotConnect: internal_connects.NewHubspotConnect(config, oauthCfg, logger, postgres),
		},
	}
}

func (connectApi *webConnectRPCApi) buildConnectParameter(c *gin.Context, provider string) (string, error) {
	// redirection after successful connect
	redirectTo, ok := c.GetQuery("redirect_to")
	if !ok {
		connectApi.logger.Errorf("connect request, there isn't any redirect url.")
		return "", errors.New("connect request, there isn't any redirect url")
	}

	// organizationId
	orgId, ok := c.GetQuery("organization_id")
	if !ok {
		connectApi.logger.Errorf("connect request, there isn't any link id to configure the link.")
		return "", errors.New("connect request, there isn't any link_id")
	}

	organizationId, err := strconv.ParseUint(orgId, 10, 64)
	if err != nil {
		connectApi.logger.Errorf("connect request, the link id is not uint 64")
		return "", err
	}

	prjId, ok := c.GetQuery("project_id")
	if !ok {
		connectApi.logger.Errorf("connect request, there isn't any link id to configure the link.")
		return "", errors.New("connect request, there isn't any link_id")
	}

	projectId, err := strconv.ParseUint(prjId, 10, 64)
	if err != nil {
		connectApi.logger.Errorf("connect request, the link id is not uint 64")
		return "", err
	}

	// connector string,
	// linker Linker, linkerId uint64, redirect string
	state, err := connectApi.notionConnect.EncodeState(
		c,
		provider,
		organizationId,
		projectId,
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
		c.Error(errors.New("ConfluenceConnect oauth2 illegal state"))
		return
	}
	url, err := connectApi.confluenceConnect.AuthCodeURL(state)
	if err != nil {
		c.Error(errors.New("ConfluenceConnect oauth2 is not enabled"))
		return
	}
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GoogleDriveConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, KN_GOOGLE_DRIVE)
	if err != nil {
		c.Error(errors.New("GoogleDriveConnect oauth2 illgal state"))
		return
	}
	url, err := connectApi.googleDriveConnect.AuthCodeURL(state)
	if err != nil {
		c.Error(errors.New("GoogleDriveConnect oauth2 is not enabled"))
		return
	}
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GithubCodeConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, KN_GITHUB_CODE)
	if err != nil {
		c.Error(errors.New("GithubCodeConnect oauth2 illgal state"))
		return
	}
	url, err := connectApi.githubCodeConnect.AuthCodeURL(state)
	if err != nil {
		c.Error(errors.New("GithubCodeConnect oauth2 is not enabled"))
		return
	}
	connectApi.logger.Debugf("url generated for github connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GitlabCodeConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, KN_GITLAB_CODE)
	if err != nil {
		c.Error(errors.New("GitlabCodeConnect oauth2 illgal state"))
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
		c.Error(errors.New("MicrosoftSharepointConnect oauth2 illgal state"))
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
		c.Error(errors.New("MicrosoftOnedriveConnect oauth2 illgal state"))
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
		c.Error(errors.New("NotionConnect oauth2 illgal state"))
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
		c.Error(errors.New("JiraActionConnect oauth2 illgal state"))
		return
	}
	url, err := connectApi.jiraConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) GmailActionConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, AN_GOOGLE_GMAIL)
	if err != nil {
		c.Error(errors.New("GmailActionConnect oauth2 illgal state"))
		return
	}
	url, err := connectApi.gmailConnect.AuthCodeURL(state)
	if err != nil {
		c.Error(errors.New("gmail oauth2 is not enabled"))
		return
	}
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) HubspotCRMConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, CRM_HUBSPOT)
	if err != nil {
		c.Error(errors.New("HubspotCRMConnect oauth2 illgal state"))
		return
	}
	url := connectApi.hubspotConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectRPCApi) SlackActionConnect(c *gin.Context) {
	state, err := connectApi.buildConnectParameter(c, AN_SLACK)
	if err != nil {
		c.Error(errors.New("SlackActionConnect oauth2 illgal state"))
		return
	}
	url := connectApi.slackConnect.AuthCodeURL(state)
	connectApi.logger.Debugf("url generated for confluence connect %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (connectApi *webConnectGRPCApi) GetConnectorFiles(ctx context.Context,
	r *protos.GetConnectorFilesRequest) (*protos.GetConnectorFilesResponse, error) {
	//
	// toolId uint64, q *string, pageToken *string
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	organizationContext, err := iAuth.OrganizationContext()
	if err != nil {
		connectApi.logger.Errorf("unable to resolve organization context for connector files: %v", err)
		return utils.AuthenticateError[protos.GetConnectorFilesResponse]()
	}

	// need to modify
	crd, err := connectApi.vaultService.GetProviderCredential(
		ctx, organizationContext.OrganizationID, r.GetProvider())
	if err != nil {
		connectApi.logger.Errorf("unable to get tool credentials %v", err)
		return utils.Error[protos.GetConnectorFilesResponse](err, "Unable to get tool credential to get list of files.")
	}
	token, connect, err := connectApi.googleDriveConnect.ToToken(crd.Value)
	if err != nil {
		connectApi.logger.Errorf("unable to get tool credentials %v", err)
		return utils.Error[protos.GetConnectorFilesResponse](err, "Unable to get tool credential to get list of files.")
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
			return utils.Error[protos.GetConnectorFilesResponse](err, "Unable to get files ")
		}

		return utils.Success[protos.GetConnectorFilesResponse](fls.Items)
	case KN_CONFLUENCE:
		fls, err := connectApi.confluenceConnect.ConfluencePages(ctx, token, q, pageToken)
		if err != nil {
			connectApi.logger.Errorf("unable to get tool credentials %v", err)
			return utils.Error[protos.GetConnectorFilesResponse](err, "Unable to get files ")
		}
		return utils.Success[protos.GetConnectorFilesResponse](fls.Items)
	case KN_NOTION:
		fls, err := connectApi.notionConnect.NotionPages(ctx, token, q, pageToken)
		if err != nil {
			connectApi.logger.Errorf("unable to get tool credentials %v", err)
			return utils.Error[protos.GetConnectorFilesResponse](err, "Unable to get files ")
		}
		return utils.Success[protos.GetConnectorFilesResponse](fls.Results)
	case KN_ONE_DRIVE:
		fls, err := connectApi.microsoftOnedriveConnect.OneDriveFiles(ctx, token, q, pageToken)
		if err != nil {
			connectApi.logger.Errorf("unable to get tool credentials %v", err)
			return utils.Error[protos.GetConnectorFilesResponse](err, "Unable to get files ")
		}
		return utils.Success[protos.GetConnectorFilesResponse](fls.Value)
	case KN_SHARE_POINT:
		fls, err := connectApi.microsoftSharepointConnect.SharePointFiles(ctx, token, q, pageToken)
		if err != nil {
			connectApi.logger.Errorf("unable to get tool credentials %v", err)
			return utils.Error[protos.GetConnectorFilesResponse](err, "Unable to get files ")
		}
		return utils.Success[protos.GetConnectorFilesResponse](fls.Value)
	case KN_GITHUB_CODE:
		fls, err := connectApi.githubCodeConnect.Repositories(ctx, token, q, pageToken)
		if err != nil {
			connectApi.logger.Errorf("unable to get tool credentials %v", err)
			return utils.Error[protos.GetConnectorFilesResponse](err, "Unable to get files ")
		}
		return utils.Success[protos.GetConnectorFilesResponse](fls)
	default:
		return utils.AuthenticateError[protos.GetConnectorFilesResponse]()
	}
}
