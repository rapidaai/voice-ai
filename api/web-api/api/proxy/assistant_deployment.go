package web_proxy_api

import (
	"context"
	"errors"

	assistant_client "github.com/rapidaai/pkg/clients/workflow"
	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"

	web_api "github.com/rapidaai/api/web-api/api"
	config "github.com/rapidaai/api/web-api/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
)

type webAssistantDeploymentApi struct {
	web_api.WebApi
	cfg             *config.WebAppConfig
	logger          commons.Logger
	postgres        connectors.PostgresConnector
	redis           connectors.RedisConnector
	assistantClient assistant_client.AssistantServiceClient
}

type webAssistantDeploymentGRPCApi struct {
	webAssistantDeploymentApi
}

// GetAssistantApiDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantApiDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.GetAssistantApiDeployment(c, iAuth, iRequest)
}

// GetAssistantDebuggerDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantDebuggerDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.GetAssistantDebuggerDeployment(c, iAuth, iRequest)
}

// GetAssistantPhoneDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantPhoneDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.GetAssistantPhoneDeployment(c, iAuth, iRequest)
}

// GetAssistantWebpluginDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantWebpluginDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.GetAssistantWebpluginDeployment(c, iAuth, iRequest)
}

// GetAssistantWhatsappDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantWhatsappDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.GetAssistantWhatsappDeployment(c, iAuth, iRequest)
}

func (w *webAssistantDeploymentGRPCApi) CreateAssistantApiDeployment(c context.Context, iRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.CreateAssistantApiDeployment(c, iAuth, iRequest)
}

// CreateAssistantDebuggerDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantDebuggerDeployment(c context.Context, iRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.CreateAssistantDebuggerDeployment(c, iAuth, iRequest)
}

// CreateAssistantPhoneDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantPhoneDeployment(c context.Context, iRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.CreateAssistantPhoneDeployment(c, iAuth, iRequest)
}

// CreateAssistantWebpluginDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantWebpluginDeployment(c context.Context, iRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.CreateAssistantWebpluginDeployment(c, iAuth, iRequest)
}

// CreateAssistantWhatsappDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantWhatsappDeployment(c context.Context, iRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	return w.assistantClient.CreateAssistantWhatsappDeployment(c, iAuth, iRequest)
}

// G
func NewAssistantDeploymentGRPCApi(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.AssistantDeploymentServiceServer {
	return &webAssistantDeploymentGRPCApi{
		webAssistantDeploymentApi{
			WebApi:          web_api.NewWebApi(config, logger, postgres, redis),
			cfg:             config,
			logger:          logger,
			postgres:        postgres,
			redis:           redis,
			assistantClient: assistant_client.NewAssistantServiceClientGRPC(&config.AppConfig, logger, redis),
		},
	}
}
