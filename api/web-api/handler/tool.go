package web_handler

import (
	"context"
	"errors"

	config "github.com/lexatic/web-backend/config"
	assistant_client "github.com/lexatic/web-backend/pkg/clients/workflow"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type webToolApi struct {
	WebApi
	cfg        *config.AppConfig
	logger     commons.Logger
	postgres   connectors.PostgresConnector
	redis      connectors.RedisConnector
	toolClient assistant_client.AssistantServiceClient
}

type webToolGRPCApi struct {
	webToolApi
}

func NewToolGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.ToolServiceServer {
	return &webToolGRPCApi{
		webToolApi{
			WebApi:     NewWebApi(config, logger, postgres, redis),
			cfg:        config,
			logger:     logger,
			postgres:   postgres,
			redis:      redis,
			toolClient: assistant_client.NewAssistantServiceClientGRPC(config, logger, redis),
		},
	}
}

func (toolApi *webToolGRPCApi) GetAllTool(ctx context.Context, cepm *web_api.GetAllToolRequest) (*web_api.GetAllToolResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		toolApi.logger.Errorf("unauthenticated request to get all tools")
		return utils.Error[web_api.GetAllToolResponse](
			errors.New("unauthenticated request for get all tool to integrate with assistant"),
			"Please provider valid service credentials to perform GetAssistantSkill, read docs @ docs.rapida.ai",
		)
	}
	page, tls, err := toolApi.toolClient.GetAllTool(ctx, iAuth, cepm.GetCriterias(), cepm.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllToolResponse](
			err,
			"Unable to get all the tools, please try again later.",
		)
	}

	return utils.PaginatedSuccess[web_api.GetAllToolResponse, []*web_api.Tool](
		page.GetTotalItem(), page.GetCurrentPage(),
		tls)
}

func (toolApi *webToolGRPCApi) GetTool(c context.Context, iRequest *web_api.GetToolRequest) (*web_api.GetToolResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		toolApi.logger.Errorf("unauthenticated request for get tool")
		return nil, errors.New("unauthenticated request")
	}
	tl, err := toolApi.toolClient.GetTool(c, iAuth, iRequest)
	if err != nil {
		return utils.Error[web_api.GetToolResponse](
			err,
			"Unable to get tool, please try again in sometime.")
	}
	return utils.Success[web_api.GetToolResponse, *web_api.Tool](tl)
}
