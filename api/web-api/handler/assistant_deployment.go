package web_handler

import (
	"context"
	"errors"

	assistant_client "github.com/lexatic/web-backend/pkg/clients/workflow"
	"github.com/lexatic/web-backend/pkg/types"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
)

type webAssistantDeploymentApi struct {
	WebApi
	cfg             *config.AppConfig
	logger          commons.Logger
	postgres        connectors.PostgresConnector
	redis           connectors.RedisConnector
	assistantClient assistant_client.AssistantServiceClient
}

type webAssistantDeploymentGRPCApi struct {
	webAssistantDeploymentApi
}

// GetAssistantApiDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantApiDeployment(c context.Context, iRequest *web_api.GetAssistantDeploymentRequest) (*web_api.GetAssistantApiDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.GetAssistantApiDeployment(c, iAuth, iRequest)
}

// GetAssistantDebuggerDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantDebuggerDeployment(c context.Context, iRequest *web_api.GetAssistantDeploymentRequest) (*web_api.GetAssistantDebuggerDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.GetAssistantDebuggerDeployment(c, iAuth, iRequest)
}

// GetAssistantPhoneDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantPhoneDeployment(c context.Context, iRequest *web_api.GetAssistantDeploymentRequest) (*web_api.GetAssistantPhoneDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.GetAssistantPhoneDeployment(c, iAuth, iRequest)
}

// GetAssistantWebpluginDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantWebpluginDeployment(c context.Context, iRequest *web_api.GetAssistantDeploymentRequest) (*web_api.GetAssistantWebpluginDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.GetAssistantWebpluginDeployment(c, iAuth, iRequest)
}

// GetAssistantWhatsappDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantWhatsappDeployment(c context.Context, iRequest *web_api.GetAssistantDeploymentRequest) (*web_api.GetAssistantWhatsappDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.GetAssistantWhatsappDeployment(c, iAuth, iRequest)
}

func (w *webAssistantDeploymentGRPCApi) CreateAssistantApiDeployment(c context.Context, iRequest *web_api.CreateAssistantDeploymentRequest) (*web_api.GetAssistantApiDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.CreateAssistantApiDeployment(c, iAuth, iRequest)
}

// CreateAssistantDebuggerDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantDebuggerDeployment(c context.Context, iRequest *web_api.CreateAssistantDeploymentRequest) (*web_api.GetAssistantDebuggerDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.CreateAssistantDebuggerDeployment(c, iAuth, iRequest)
}

// CreateAssistantPhoneDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantPhoneDeployment(c context.Context, iRequest *web_api.CreateAssistantDeploymentRequest) (*web_api.GetAssistantPhoneDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.CreateAssistantPhoneDeployment(c, iAuth, iRequest)
}

// CreateAssistantWebpluginDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantWebpluginDeployment(c context.Context, iRequest *web_api.CreateAssistantDeploymentRequest) (*web_api.GetAssistantWebpluginDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.CreateAssistantWebpluginDeployment(c, iAuth, iRequest)
}

// CreateAssistantWhatsappDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantWhatsappDeployment(c context.Context, iRequest *web_api.CreateAssistantDeploymentRequest) (*web_api.GetAssistantWhatsappDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.CreateAssistantWhatsappDeployment(c, iAuth, iRequest)
}

// G
func NewAssistantDeploymentGRPCApi(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.AssistantDeploymentServiceServer {
	return &webAssistantDeploymentGRPCApi{
		webAssistantDeploymentApi{
			WebApi:          NewWebApi(config, logger, postgres, redis),
			cfg:             config,
			logger:          logger,
			postgres:        postgres,
			redis:           redis,
			assistantClient: assistant_client.NewAssistantServiceClientGRPC(config, logger, redis),
		},
	}
}
