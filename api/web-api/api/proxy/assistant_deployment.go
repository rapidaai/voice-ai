package web_proxy_api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	assistant_client "github.com/rapidaai/pkg/clients/workflow"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
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

// GetAssistantApiDeployment implements protos.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantApiDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.GetAssistantApiDeployment(c, principal, iRequest)
}

func (w *webAssistantDeploymentGRPCApi) GetAllAssistantApiDeployment(c context.Context, iRequest *protos.GetAllAssistantDeploymentRequest) (*protos.GetAllAssistantApiDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	page, data, err := w.assistantClient.GetAllAssistantApiDeployment(c, principal, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[protos.GetAllAssistantApiDeploymentResponse, []*protos.AssistantApiDeployment](
		page.GetTotalItem(), page.GetCurrentPage(),
		data)
}

func (w *webAssistantDeploymentGRPCApi) DisableAssistantApiDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.DisableAssistantApiDeployment(c, principal, iRequest)
}

// GetAssistantDebuggerDeployment implements protos.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantDebuggerDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.GetAssistantDebuggerDeployment(c, principal, iRequest)
}

func (w *webAssistantDeploymentGRPCApi) GetAllAssistantDebuggerDeployment(c context.Context, iRequest *protos.GetAllAssistantDeploymentRequest) (*protos.GetAllAssistantDebuggerDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	page, data, err := w.assistantClient.GetAllAssistantDebuggerDeployment(c, principal, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[protos.GetAllAssistantDebuggerDeploymentResponse, []*protos.AssistantDebuggerDeployment](
		page.GetTotalItem(), page.GetCurrentPage(),
		data)
}

func (w *webAssistantDeploymentGRPCApi) DisableAssistantDebuggerDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.DisableAssistantDebuggerDeployment(c, principal, iRequest)
}

// GetAssistantPhoneDeployment implements protos.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantPhoneDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.GetAssistantPhoneDeployment(c, principal, iRequest)
}

func (w *webAssistantDeploymentGRPCApi) GetAllAssistantPhoneDeployment(c context.Context, iRequest *protos.GetAllAssistantDeploymentRequest) (*protos.GetAllAssistantPhoneDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	page, data, err := w.assistantClient.GetAllAssistantPhoneDeployment(c, principal, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[protos.GetAllAssistantPhoneDeploymentResponse, []*protos.AssistantPhoneDeployment](
		page.GetTotalItem(), page.GetCurrentPage(),
		data)
}

func (w *webAssistantDeploymentGRPCApi) DisableAssistantPhoneDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.DisableAssistantPhoneDeployment(c, principal, iRequest)
}

// GetAssistantWebpluginDeployment implements protos.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantWebpluginDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.GetAssistantWebpluginDeployment(c, principal, iRequest)
}

func (w *webAssistantDeploymentGRPCApi) GetAllAssistantWebpluginDeployment(c context.Context, iRequest *protos.GetAllAssistantDeploymentRequest) (*protos.GetAllAssistantWebpluginDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	page, data, err := w.assistantClient.GetAllAssistantWebpluginDeployment(c, principal, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[protos.GetAllAssistantWebpluginDeploymentResponse, []*protos.AssistantWebpluginDeployment](
		page.GetTotalItem(), page.GetCurrentPage(),
		data)
}

func (w *webAssistantDeploymentGRPCApi) DisableAssistantWebpluginDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.DisableAssistantWebpluginDeployment(c, principal, iRequest)
}

// GetAssistantWhatsappDeployment implements protos.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) GetAssistantWhatsappDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.GetAssistantWhatsappDeployment(c, principal, iRequest)
}

func (w *webAssistantDeploymentGRPCApi) GetAllAssistantWhatsappDeployment(c context.Context, iRequest *protos.GetAllAssistantDeploymentRequest) (*protos.GetAllAssistantWhatsappDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	page, data, err := w.assistantClient.GetAllAssistantWhatsappDeployment(c, principal, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[protos.GetAllAssistantWhatsappDeploymentResponse, []*protos.AssistantWhatsappDeployment](
		page.GetTotalItem(), page.GetCurrentPage(),
		data)
}

func (w *webAssistantDeploymentGRPCApi) DisableAssistantWhatsappDeployment(c context.Context, iRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.DisableAssistantWhatsappDeployment(c, principal, iRequest)
}

func (w *webAssistantDeploymentGRPCApi) CreateAssistantApiDeployment(c context.Context, iRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.CreateAssistantApiDeployment(c, principal, iRequest)
}

// CreateAssistantDebuggerDeployment implements protos.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantDebuggerDeployment(c context.Context, iRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.CreateAssistantDebuggerDeployment(c, principal, iRequest)
}

// CreateAssistantPhoneDeployment implements protos.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantPhoneDeployment(c context.Context, iRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.CreateAssistantPhoneDeployment(c, principal, iRequest)
}

// CreateAssistantWebpluginDeployment implements protos.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantWebpluginDeployment(c context.Context, iRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.CreateAssistantWebpluginDeployment(c, principal, iRequest)
}

// CreateAssistantWhatsappDeployment implements protos.AssistantDeploymentServiceServer.
func (w *webAssistantDeploymentGRPCApi) CreateAssistantWhatsappDeployment(c context.Context, iRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return w.assistantClient.CreateAssistantWhatsappDeployment(c, principal, iRequest)
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
