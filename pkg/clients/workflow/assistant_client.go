// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package workflow_client

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	clients "github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

type AssistantServiceClient interface {
	GetAllAssistant(c context.Context, auth types.SimplePrinciple, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error)

	DeleteAssistant(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.DeleteAssistantRequest) (*protos.GetAssistantResponse, error)
	GetAssistant(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error)
	CreateAssistant(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantRequest) (*protos.GetAssistantResponse, error)

	GetAllAssistantProvider(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.GetAllAssistantProviderResponse_AssistantProvider, error)
	UpdateAssistantVersion(c context.Context, auth types.SimplePrinciple, iRequest *protos.UpdateAssistantVersionRequest) (*protos.GetAssistantResponse, error)
	CreateAssistantProvider(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantProviderRequest) (*protos.GetAssistantProviderResponse, error)

	//
	GetAllMessage(c context.Context, auth types.SimplePrinciple,
		criteria []*protos.Criteria, paginate *protos.Paginate,
		order *protos.Ordering, selectors []*protos.FieldSelector) (*protos.Paginated, []*protos.AssistantConversationMessage, error)
	GetAllAssistantMessage(c context.Context, auth types.SimplePrinciple, assistantId uint64,
		criteria []*protos.Criteria, paginate *protos.Paginate,
		order *protos.Ordering, selectors []*protos.FieldSelector) (*protos.Paginated, []*protos.AssistantConversationMessage, error)
	GetAllAssistantConversation(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate, order *protos.Ordering) (*protos.Paginated, []*protos.AssistantConversation, error)
	GetAllConversationMessage(ctx context.Context, auth types.SimplePrinciple, assistantId, assistantConversationId uint64, criteria []*protos.Criteria, paginate *protos.Paginate, order *protos.Ordering) (*protos.Paginated, []*protos.AssistantConversationMessage, error)
	GetAssistantDashboard(ctx context.Context, auth types.SimplePrinciple, dashboardRequest *protos.GetAssistantDashboardRequest) (*protos.GetAssistantDashboardResponse, error)
	GetAssistantConversation(
		c context.Context,
		auth types.SimplePrinciple,
		assistantRequest *protos.GetAssistantConversationRequest) (*protos.GetAssistantConversationResponse, error)

	CreateAssistantTag(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantTagRequest) (*protos.GetAssistantResponse, error)
	UpdateAssistantDetail(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.UpdateAssistantDetailRequest) (*protos.GetAssistantResponse, error)

	// deployment
	CreateAssistantApiDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error)
	CreateAssistantPhoneDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error)
	CreateAssistantWhatsappDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error)
	CreateAssistantWebpluginDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error)
	CreateAssistantDebuggerDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error)

	GetAssistantApiDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error)
	GetAssistantPhoneDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error)
	GetAssistantWhatsappDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error)
	GetAssistantWebpluginDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error)
	GetAssistantDebuggerDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error)
	GetAllAssistantApiDeployment(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantApiDeployment, error)
	GetAllAssistantPhoneDeployment(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantPhoneDeployment, error)
	GetAllAssistantWhatsappDeployment(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantWhatsappDeployment, error)
	GetAllAssistantWebpluginDeployment(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantWebpluginDeployment, error)
	GetAllAssistantDebuggerDeployment(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantDebuggerDeployment, error)
	DisableAssistantApiDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error)
	DisableAssistantPhoneDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error)
	DisableAssistantWhatsappDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error)
	DisableAssistantWebpluginDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error)
	DisableAssistantDebuggerDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error)

	//
	GetAssistantHTTPLog(ctx context.Context, auth types.SimplePrinciple, req *protos.GetAssistantHTTPLogRequest) (*protos.GetAssistantHTTPLogResponse, error)
	GetAllAssistantHTTPLog(ctx context.Context, auth types.SimplePrinciple, projectId uint64, criteria []*protos.Criteria, paginate *protos.Paginate, ordering *protos.Ordering) (*protos.Paginated, []*protos.AssistantHTTPLog, error)
	RetryAssistantHTTPLog(ctx context.Context, auth types.SimplePrinciple, req *protos.RetryAssistantHTTPLogRequest) (*protos.GetAssistantHTTPLogResponse, error)

	//
	GetAllAssistantTool(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantTool, error)
	GetAssistantTool(c context.Context, auth types.SimplePrinciple, iRequest *protos.GetAssistantToolRequest) (*protos.GetAssistantToolResponse, error)
	CreateAssistantTool(c context.Context, auth types.SimplePrinciple, iRequest *protos.CreateAssistantToolRequest) (*protos.GetAssistantToolResponse, error)
	UpdateAssistantTool(c context.Context, auth types.SimplePrinciple, iRequest *protos.UpdateAssistantToolRequest) (*protos.GetAssistantToolResponse, error)
	DeleteAssistantTool(c context.Context, auth types.SimplePrinciple, iRequest *protos.DeleteAssistantToolRequest) (*protos.GetAssistantToolResponse, error)

	//
	GetAllAssistantKnowledge(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantKnowledge, error)
	GetAssistantKnowledge(c context.Context, auth types.SimplePrinciple, iRequest *protos.GetAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error)
	CreateAssistantKnowledge(c context.Context, auth types.SimplePrinciple, iRequest *protos.CreateAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error)
	UpdateAssistantKnowledge(c context.Context, auth types.SimplePrinciple, iRequest *protos.UpdateAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error)
	DeleteAssistantKnowledge(c context.Context, auth types.SimplePrinciple, iRequest *protos.DeleteAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error)

	GetAssistantToolLog(ctx context.Context, auth types.SimplePrinciple, in *protos.GetAssistantToolLogRequest) (*protos.GetAssistantToolLogResponse, error)
	GetAllAssistantToolLog(ctx context.Context, auth types.SimplePrinciple, in *protos.GetAllAssistantToolLogRequest) (*protos.GetAllAssistantToolLogResponse, error)

	// assistant configurations
	GetAssistantConfiguration(ctx context.Context, auth types.SimplePrinciple, in *protos.GetAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error)
	GetAllAssistantConfiguration(ctx context.Context, auth types.SimplePrinciple, in *protos.GetAllAssistantConfigurationRequest) (*protos.GetAllAssistantConfigurationResponse, error)
	CreateAssistantConfiguration(ctx context.Context, auth types.SimplePrinciple, in *protos.CreateAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error)
	UpdateAssistantConfiguration(ctx context.Context, auth types.SimplePrinciple, in *protos.UpdateAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error)
	DeleteAssistantConfiguration(ctx context.Context, auth types.SimplePrinciple, in *protos.DeleteAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error)
}

type assistantServiceClient struct {
	clients.InternalClient
	cfg                       *config.AppConfig
	logger                    commons.Logger
	assistantClient           protos.AssistantServiceClient
	assistantDeploymentClient protos.AssistantDeploymentServiceClient
}

// NewAssistantServiceClientGRPC creates a new instance of AssistantServiceClient using gRPC.
// It establishes a connection to the assistant service using the provided configuration, logger, and Redis connector.
//
// Parameters:
// - config: The application configuration containing the workflow host details.
// - logger: A Logger instance for logging messages.
// - redis: A RedisConnector instance for connecting to Redis.
//
// Returns:
// - An instance of AssistantServiceClient, or nil if an error occurs during connection establishment.
func NewAssistantServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) AssistantServiceClient {
	conn, err := grpc.NewClient(config.Assistant.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Errorf("Unable to create connection %v", err)
	}
	return &assistantServiceClient{
		cfg:                       config,
		logger:                    logger,
		InternalClient:            clients.NewInternalClient(config, logger, redis),
		assistantClient:           protos.NewAssistantServiceClient(conn),
		assistantDeploymentClient: protos.NewAssistantDeploymentServiceClient(conn),
	}
}

func (client *assistantServiceClient) GetAllAssistant(c context.Context, auth types.SimplePrinciple, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error) {
	res, err := client.assistantClient.GetAllAssistant(client.WithAuth(c, auth), &protos.GetAllAssistantRequest{
		Paginate:  paginate,
		Criterias: criteria,
	})
	if err != nil {
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all assistant %v", err)
	}

	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) DeleteAssistant(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.DeleteAssistantRequest) (*protos.GetAssistantResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.DeleteAssistant(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.DeleteAssistant", time.Since(start))
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to delete assistant %v", err)
	}
	return res, nil
}

func (client *assistantServiceClient) GetAssistant(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAssistant(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAssistant", time.Since(start))
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get assistant %v", err)
	}
	return res, nil
}

func (client *assistantServiceClient) CreateAssistant(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantRequest) (*protos.GetAssistantResponse, error) {
	res, err := client.assistantClient.CreateAssistant(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateAssistant %v", err)
		return nil, err
	}
	return res, nil
}

func (client *assistantServiceClient) GetAllAssistantProvider(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.GetAllAssistantProviderResponse_AssistantProvider, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAllAssistantProvider(client.WithAuth(c, auth), &protos.GetAllAssistantProviderRequest{
		Criterias:   criteria,
		Paginate:    paginate,
		AssistantId: assistantId,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAllAssistantProvider", time.Since(start))
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all assistant %v", err)
	}

	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) UpdateAssistantVersion(c context.Context, auth types.SimplePrinciple, request *protos.UpdateAssistantVersionRequest) (*protos.GetAssistantResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.UpdateAssistantVersion(client.WithAuth(c, auth), request)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.UpdateAssistantVersion", time.Since(start))
		client.logger.Errorf("error while calling to UpdateAssistantVersion %v", err)
		return nil, err
	}
	return res, nil
}

func (client *assistantServiceClient) CreateAssistantProvider(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantProviderRequest) (*protos.GetAssistantProviderResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.CreateAssistantProvider(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.CreateAssistantProvider", time.Since(start))
		client.logger.Errorf("error while calling to CreateAssistantProvider %v", err)
		return nil, err
	}
	return res, nil
}

func (client *assistantServiceClient) CreateAssistantTag(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantTagRequest) (*protos.GetAssistantResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.CreateAssistantTag(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.UpdateAssistantDetail", time.Since(start))
		client.logger.Errorf("error while calling CreateAssistantTag %v", err)
		return nil, err
	}
	return res, nil
}

func (client *assistantServiceClient) UpdateAssistantDetail(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.UpdateAssistantDetailRequest) (*protos.GetAssistantResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.UpdateAssistantDetail(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.UpdateAssistantDetail", time.Since(start))
		client.logger.Errorf("error while calling CreateAssistantTag %v", err)
		return nil, err
	}
	return res, nil
}

func (client *assistantServiceClient) GetAllMessage(ctx context.Context,
	auth types.SimplePrinciple,
	criteria []*protos.Criteria,
	paginate *protos.Paginate,
	order *protos.Ordering,
	fieldSelector []*protos.FieldSelector,
) (*protos.Paginated, []*protos.AssistantConversationMessage, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAllMessage(client.WithAuth(ctx, auth), &protos.GetAllMessageRequest{
		Paginate:  paginate,
		Criterias: criteria,
		Order:     order,
		Selectors: fieldSelector,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAllMessage", time.Since(start))
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAllMessage", time.Since(start))
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAllMessage", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) GetAllAssistantMessage(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64, criteria []*protos.Criteria,
	paginate *protos.Paginate,
	order *protos.Ordering,
	fieldSelector []*protos.FieldSelector,
) (*protos.Paginated, []*protos.AssistantConversationMessage, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAllAssistantMessage(client.WithAuth(ctx, auth), &protos.GetAllAssistantMessageRequest{
		AssistantId: assistantId,
		Paginate:    paginate,
		Criterias:   criteria,
		Order:       order,
		Selectors:   fieldSelector,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAllAssistantMessage", time.Since(start))
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all assistant %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAllAssistantMessage", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) GetAllAssistantConversation(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate, order *protos.Ordering) (*protos.Paginated, []*protos.AssistantConversation, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAllAssistantConversation(client.WithAuth(ctx, auth), &protos.GetAllAssistantConversationRequest{
		AssistantId: assistantId,
		Paginate:    paginate,
		Criterias:   criteria,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAllAssistantConversation", time.Since(start))
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all assistant %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAllAssistantConversation", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) GetAllConversationMessage(ctx context.Context, auth types.SimplePrinciple, assistantId, assistantConversationId uint64, criteria []*protos.Criteria, paginate *protos.Paginate, order *protos.Ordering) (*protos.Paginated, []*protos.AssistantConversationMessage, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAllConversationMessage(client.WithAuth(ctx, auth), &protos.GetAllConversationMessageRequest{
		AssistantConversationId: assistantConversationId,
		AssistantId:             assistantId,
		Paginate:                paginate,
		Criterias:               criteria,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAllConversationMessage", time.Since(start))
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all assistant %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAllConversationMessage", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) GetAssistantDashboard(ctx context.Context, auth types.SimplePrinciple, dashboardRequest *protos.GetAssistantDashboardRequest) (*protos.GetAssistantDashboardResponse, error) {
	start := time.Now()
	response, err := client.assistantClient.GetAssistantDashboard(client.WithAuth(ctx, auth), dashboardRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAssistantDashboard", time.Since(start))
		client.logger.Errorf("error while calling get assistant dashboard %v", err)
		return nil, err
	}
	if !response.GetSuccess() {
		client.logger.Errorf("error while calling get assistant dashboard %v", response.GetError())
	}
	client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAssistantDashboard", time.Since(start))
	return response, nil
}

func (client *assistantServiceClient) CreateAssistantApiDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.CreateAssistantApiDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.CreateAssistantApiDeployment", time.Since(start))
		client.logger.Errorf("error while calling CreateAssistantApiDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.CreateAssistantApiDeployment", time.Since(start))
	return res, nil
}
func (client *assistantServiceClient) CreateAssistantPhoneDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.CreateAssistantPhoneDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.CreateAssistantPhoneDeployment", time.Since(start))
		client.logger.Errorf("error while calling CreateAssistantPhoneDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantServiceClient.CreateAssistantPhoneDeployment", time.Since(start))
	return res, nil
}
func (client *assistantServiceClient) CreateAssistantWhatsappDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.CreateAssistantWhatsappDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.CreateAssistantWhatsappDeployment", time.Since(start))
		client.logger.Errorf("error while calling CreateAssistantWhatsappDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.CreateAssistantWhatsappDeployment", time.Since(start))
	return res, nil
}
func (client *assistantServiceClient) CreateAssistantWebpluginDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.CreateAssistantWebpluginDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.CreateAssistantWebpluginDeployment", time.Since(start))
		client.logger.Errorf("error while calling CreateAssistantWebpluginDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.CreateAssistantWebpluginDeployment", time.Since(start))
	return res, nil
}
func (client *assistantServiceClient) CreateAssistantDebuggerDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.CreateAssistantDebuggerDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.CreateAssistantDebuggerDeployment", time.Since(start))
		client.logger.Errorf("error while calling CreateAssistantDebuggerDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.CreateAssistantDebuggerDeployment", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) GetAssistantApiDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.GetAssistantApiDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAssistantApiDeployment", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantApiDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.CreateAssistantDebuggerDeployment", time.Since(start))
	return res, nil
}
func (client *assistantServiceClient) GetAssistantPhoneDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.GetAssistantPhoneDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAssistantPhoneDeployment", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantPhoneDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAssistantPhoneDeployment", time.Since(start))
	return res, nil
}
func (client *assistantServiceClient) GetAssistantWhatsappDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.GetAssistantWhatsappDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAssistantWhatsappDeployment", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantWhatsappDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAssistantWhatsappDeployment", time.Since(start))
	return res, nil
}
func (client *assistantServiceClient) GetAssistantWebpluginDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.GetAssistantWebpluginDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAssistantWebpluginDeployment", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantWebpluginDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAssistantWebpluginDeployment", time.Since(start))
	client.logger.Debugf("report %+v", res.Data)
	return res, nil
}
func (client *assistantServiceClient) GetAssistantDebuggerDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.GetAssistantDebuggerDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAssistantDebuggerDeployment", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantDebuggerDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAssistantDebuggerDeployment", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) GetAllAssistantApiDeployment(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantApiDeployment, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.GetAllAssistantApiDeployment(client.WithAuth(c, auth), &protos.GetAllAssistantDeploymentRequest{
		AssistantId: assistantId,
		Paginate:    paginate,
		Criterias:   criteria,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAllAssistantApiDeployment", time.Since(start))
		client.logger.Errorf("error while calling GetAllAssistantApiDeployment %v", err)
		return nil, nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAllAssistantApiDeployment", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) GetAllAssistantPhoneDeployment(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantPhoneDeployment, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.GetAllAssistantPhoneDeployment(client.WithAuth(c, auth), &protos.GetAllAssistantDeploymentRequest{
		AssistantId: assistantId,
		Paginate:    paginate,
		Criterias:   criteria,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAllAssistantPhoneDeployment", time.Since(start))
		client.logger.Errorf("error while calling GetAllAssistantPhoneDeployment %v", err)
		return nil, nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAllAssistantPhoneDeployment", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) GetAllAssistantWhatsappDeployment(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantWhatsappDeployment, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.GetAllAssistantWhatsappDeployment(client.WithAuth(c, auth), &protos.GetAllAssistantDeploymentRequest{
		AssistantId: assistantId,
		Paginate:    paginate,
		Criterias:   criteria,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAllAssistantWhatsappDeployment", time.Since(start))
		client.logger.Errorf("error while calling GetAllAssistantWhatsappDeployment %v", err)
		return nil, nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAllAssistantWhatsappDeployment", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) GetAllAssistantWebpluginDeployment(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantWebpluginDeployment, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.GetAllAssistantWebpluginDeployment(client.WithAuth(c, auth), &protos.GetAllAssistantDeploymentRequest{
		AssistantId: assistantId,
		Paginate:    paginate,
		Criterias:   criteria,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAllAssistantWebpluginDeployment", time.Since(start))
		client.logger.Errorf("error while calling GetAllAssistantWebpluginDeployment %v", err)
		return nil, nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAllAssistantWebpluginDeployment", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) GetAllAssistantDebuggerDeployment(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantDebuggerDeployment, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.GetAllAssistantDebuggerDeployment(client.WithAuth(c, auth), &protos.GetAllAssistantDeploymentRequest{
		AssistantId: assistantId,
		Paginate:    paginate,
		Criterias:   criteria,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAllAssistantDebuggerDeployment", time.Since(start))
		client.logger.Errorf("error while calling GetAllAssistantDebuggerDeployment %v", err)
		return nil, nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.GetAllAssistantDebuggerDeployment", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) DisableAssistantApiDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantApiDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.DisableAssistantApiDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.DisableAssistantApiDeployment", time.Since(start))
		client.logger.Errorf("error while calling DisableAssistantApiDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.DisableAssistantApiDeployment", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) DisableAssistantPhoneDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.DisableAssistantPhoneDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.DisableAssistantPhoneDeployment", time.Since(start))
		client.logger.Errorf("error while calling DisableAssistantPhoneDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.DisableAssistantPhoneDeployment", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) DisableAssistantWhatsappDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWhatsappDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.DisableAssistantWhatsappDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.DisableAssistantWhatsappDeployment", time.Since(start))
		client.logger.Errorf("error while calling DisableAssistantWhatsappDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.DisableAssistantWhatsappDeployment", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) DisableAssistantWebpluginDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.DisableAssistantWebpluginDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.DisableAssistantWebpluginDeployment", time.Since(start))
		client.logger.Errorf("error while calling DisableAssistantWebpluginDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.DisableAssistantWebpluginDeployment", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) DisableAssistantDebuggerDeployment(c context.Context, auth types.SimplePrinciple, assistantRequest *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantDebuggerDeploymentResponse, error) {
	start := time.Now()
	res, err := client.assistantDeploymentClient.DisableAssistantDebuggerDeployment(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantDeploymentClient.DisableAssistantDebuggerDeployment", time.Since(start))
		client.logger.Errorf("error while calling DisableAssistantDebuggerDeployment %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantDeploymentClient.DisableAssistantDebuggerDeployment", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) GetAllAssistantHTTPLog(ctx context.Context, auth types.SimplePrinciple,
	projectId uint64,
	criteria []*protos.Criteria, paginate *protos.Paginate, ordering *protos.Ordering) (*protos.Paginated, []*protos.AssistantHTTPLog, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAllAssistantHTTPLog(client.WithAuth(ctx, auth), &protos.GetAllAssistantHTTPLogRequest{
		ProjectId: projectId,
		Paginate:  paginate,
		Criterias: criteria,
		Order:     ordering,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.GetAllAssistantHTTPLog", time.Since(start))
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all assistant %v", err)
	}

	client.logger.Benchmark("Benchmarking: assistantClient.GetAllAssistantHTTPLog", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}
func (client *assistantServiceClient) GetAssistantHTTPLog(c context.Context,
	auth types.SimplePrinciple, iRequest *protos.GetAssistantHTTPLogRequest) (*protos.GetAssistantHTTPLogResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAssistantHTTPLog(client.WithAuth(c, auth), iRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantHTTPLog", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantHTTPLog %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get GetAssistantHTTPLog %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantHTTPLog", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) RetryAssistantHTTPLog(c context.Context,
	auth types.SimplePrinciple, iRequest *protos.RetryAssistantHTTPLogRequest) (*protos.GetAssistantHTTPLogResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.RetryAssistantHTTPLog(client.WithAuth(c, auth), iRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.RetryAssistantHTTPLog", time.Since(start))
		client.logger.Errorf("error while calling RetryAssistantHTTPLog %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to retry RetryAssistantHTTPLog %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.RetryAssistantHTTPLog", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) GetAssistantConversation(
	c context.Context,
	auth types.SimplePrinciple,
	assistantRequest *protos.GetAssistantConversationRequest) (*protos.GetAssistantConversationResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAssistantConversation(client.WithAuth(c, auth), assistantRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantConversation", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantConversation %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get tool %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantConversation", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) GetAllAssistantTool(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantTool, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAllAssistantTool(client.WithAuth(c, auth), &protos.GetAllAssistantToolRequest{
		AssistantId: assistantId,
		Paginate:    paginate,
		Criterias:   criteria,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAssistantTool", time.Since(start))
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all assistant %v", err)
	}

	client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAssistantTool", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) GetAssistantTool(c context.Context,
	auth types.SimplePrinciple, iRequest *protos.GetAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAssistantTool(client.WithAuth(c, auth), iRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantTool", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantTool %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get GetAssistantTool %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantTool", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) CreateAssistantTool(c context.Context, auth types.SimplePrinciple, iRequest *protos.CreateAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.CreateAssistantTool(client.WithAuth(c, auth), iRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.CreateAssistantTool", time.Since(start))
		client.logger.Errorf("error while calling CreateAssistantTool %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get tool %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.CreateAssistantTool", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) DeleteAssistantTool(c context.Context, auth types.SimplePrinciple, iRequest *protos.DeleteAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.DeleteAssistantTool(client.WithAuth(c, auth), iRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.DeleteAssistantTool", time.Since(start))
		client.logger.Errorf("error while calling DeleteAssistantTool %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get tool %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.DeleteAssistantTool", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) UpdateAssistantTool(c context.Context, auth types.SimplePrinciple, iRequest *protos.UpdateAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.UpdateAssistantTool(client.WithAuth(c, auth), iRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.UpdateAssistantTool", time.Since(start))
		client.logger.Errorf("error while calling UpdateAssistantTool %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get tool %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.UpdateAssistantTool", time.Since(start))
	return res, nil
}

//

func (client *assistantServiceClient) GetAllAssistantKnowledge(c context.Context, auth types.SimplePrinciple, assistantId uint64, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.AssistantKnowledge, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAllAssistantKnowledge(client.WithAuth(c, auth), &protos.GetAllAssistantKnowledgeRequest{
		AssistantId: assistantId,
		Paginate:    paginate,
		Criterias:   criteria,
	})
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAssistantKnowledge", time.Since(start))
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all assistant %v", err)
	}

	client.logger.Benchmark("Benchmarking: assistantServiceClient.GetAssistantKnowledge", time.Since(start))
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantServiceClient) GetAssistantKnowledge(c context.Context,
	auth types.SimplePrinciple, iRequest *protos.GetAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAssistantKnowledge(client.WithAuth(c, auth), iRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantKnowledge", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantKnowledge %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get GetAssistantKnowledge %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantKnowledge", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) CreateAssistantKnowledge(c context.Context, auth types.SimplePrinciple, iRequest *protos.CreateAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.CreateAssistantKnowledge(client.WithAuth(c, auth), iRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.CreateAssistantKnowledge", time.Since(start))
		client.logger.Errorf("error while calling CreateAssistantKnowledge %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get tool %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.CreateAssistantKnowledge", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) DeleteAssistantKnowledge(c context.Context, auth types.SimplePrinciple, iRequest *protos.DeleteAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.DeleteAssistantKnowledge(client.WithAuth(c, auth), iRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.DeleteAssistantKnowledge", time.Since(start))
		client.logger.Errorf("error while calling DeleteAssistantKnowledge %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get tool %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.DeleteAssistantKnowledge", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) UpdateAssistantKnowledge(c context.Context, auth types.SimplePrinciple, iRequest *protos.UpdateAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.UpdateAssistantKnowledge(client.WithAuth(c, auth), iRequest)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.UpdateAssistantKnowledge", time.Since(start))
		client.logger.Errorf("error while calling UpdateAssistantKnowledge %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get tool %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.UpdateAssistantKnowledge", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) GetAssistantToolLog(c context.Context, auth types.SimplePrinciple, in *protos.GetAssistantToolLogRequest) (*protos.GetAssistantToolLogResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAssistantToolLog(client.WithAuth(c, auth), in)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantToolLog", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantToolLog %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get tool %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantToolLog", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) GetAllAssistantToolLog(ctx context.Context, auth types.SimplePrinciple, in *protos.GetAllAssistantToolLogRequest) (*protos.GetAllAssistantToolLogResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAllAssistantToolLog(client.WithAuth(ctx, auth), in)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.GetAllAssistantToolLog", time.Since(start))
		client.logger.Errorf("error while calling GetAllAssistantToolLog %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get tool %v", err)
	}
	client.logger.Benchmark("Benchmarking: assistantClient.GetAllAssistantToolLog", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) GetAssistantConfiguration(ctx context.Context, auth types.SimplePrinciple, in *protos.GetAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAssistantConfiguration(client.WithAuth(ctx, auth), in)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantConfiguration", time.Since(start))
		client.logger.Errorf("error while calling GetAssistantConfiguration %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantClient.GetAssistantConfiguration", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) GetAllAssistantConfiguration(ctx context.Context, auth types.SimplePrinciple, in *protos.GetAllAssistantConfigurationRequest) (*protos.GetAllAssistantConfigurationResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.GetAllAssistantConfiguration(client.WithAuth(ctx, auth), in)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.GetAllAssistantConfiguration", time.Since(start))
		client.logger.Errorf("error while calling GetAllAssistantConfiguration %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantClient.GetAllAssistantConfiguration", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) CreateAssistantConfiguration(ctx context.Context, auth types.SimplePrinciple, in *protos.CreateAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.CreateAssistantConfiguration(client.WithAuth(ctx, auth), in)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.CreateAssistantConfiguration", time.Since(start))
		client.logger.Errorf("error while calling CreateAssistantConfiguration %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantClient.CreateAssistantConfiguration", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) UpdateAssistantConfiguration(ctx context.Context, auth types.SimplePrinciple, in *protos.UpdateAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.UpdateAssistantConfiguration(client.WithAuth(ctx, auth), in)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.UpdateAssistantConfiguration", time.Since(start))
		client.logger.Errorf("error while calling UpdateAssistantConfiguration %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantClient.UpdateAssistantConfiguration", time.Since(start))
	return res, nil
}

func (client *assistantServiceClient) DeleteAssistantConfiguration(ctx context.Context, auth types.SimplePrinciple, in *protos.DeleteAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error) {
	start := time.Now()
	res, err := client.assistantClient.DeleteAssistantConfiguration(client.WithAuth(ctx, auth), in)
	if err != nil {
		client.logger.Benchmark("Benchmarking: assistantClient.DeleteAssistantConfiguration", time.Since(start))
		client.logger.Errorf("error while calling DeleteAssistantConfiguration %v", err)
		return nil, err
	}
	client.logger.Benchmark("Benchmarking: assistantClient.DeleteAssistantConfiguration", time.Since(start))
	return res, nil
}
