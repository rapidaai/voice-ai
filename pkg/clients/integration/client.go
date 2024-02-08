package integration_client

import (
	"context"
	"math"
	"strings"

	"github.com/lexatic/web-backend/config"
	clients "github.com/lexatic/web-backend/pkg/clients"
	clients_pogos "github.com/lexatic/web-backend/pkg/clients/pogos"
	commons "github.com/lexatic/web-backend/pkg/commons"
	integration_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type integrationServiceClient struct {
	cfg                *config.AppConfig
	logger             commons.Logger
	cohereClient       integration_api.CohereServiceClient
	replicateClient    integration_api.ReplicateServiceClient
	openAiClient       integration_api.OpenAiServiceClient
	anthropicClient    integration_api.AnthropicServiceClient
	googleClient       integration_api.GoogleServiceClient
	sendgridClient     integration_api.SendgridServiceClient
	auditLoggingClient integration_api.AuditLoggingServiceClient
}

func NewIntegrationServiceClientGRPC(config *config.AppConfig, logger commons.Logger) clients.IntegrationServiceClient {
	logger.Debugf("conntecting to integration client with %s", config.IntegrationHost)

	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt64),
			grpc.MaxCallSendMsgSize(math.MaxInt64),
		),
	}
	conn, err := grpc.Dial(config.IntegrationHost,
		grpcOpts...)

	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	cohereClient := integration_api.NewCohereServiceClient(conn)
	replicateClient := integration_api.NewReplicateServiceClient(conn)
	openAiClient := integration_api.NewOpenAiServiceClient(conn)
	anthropicClient := integration_api.NewAnthropicServiceClient(conn)
	googleClient := integration_api.NewGoogleServiceClient(conn)
	return &integrationServiceClient{
		cfg:                config,
		logger:             logger,
		sendgridClient:     integration_api.NewSendgridServiceClient(conn),
		auditLoggingClient: integration_api.NewAuditLoggingServiceClient(conn),
		cohereClient:       cohereClient,
		replicateClient:    replicateClient,
		openAiClient:       openAiClient,
		anthropicClient:    anthropicClient,
		googleClient:       googleClient,
	}
}

func (client *integrationServiceClient) Converse(c context.Context, request *clients_pogos.RequestData[[]*clients_pogos.Interaction]) (*integration_api.ChatResponse, error) {
	switch providerName := strings.ToLower(request.ProviderName); providerName {
	case "cohere":
		return client.converseWithCohere(c, request)
	case "anthropic":
		return client.converseWithAnthropic(c, request)
	case "replicate":
		return client.converseWithReplicate(c, request)
	case "google":
		return client.converseWithGoogle(c, request)
	default:
		return client.converseWithOpenAI(c, request)
	}
}
func (client *integrationServiceClient) Prompt(c context.Context, request *clients_pogos.RequestData[string]) (*integration_api.GenerateResponse, error) {
	// client.logger.Debugf("metadata for request %+v", )
	switch providerName := strings.ToLower(request.ProviderName); providerName {
	case "cohere":
		return client.promptCohere(c, request)
	case "anthropic":
		return client.promptAnthropic(c, request)
	case "replicate":
		return client.promptReplicate(c, request)
	case "google":
		return client.promptGoogle(c, request)
	default:
		return client.promptOpenAI(c, request)
	}

}
func (client *integrationServiceClient) GenerateTextToImage(c context.Context, request *clients_pogos.RequestData[string]) (*integration_api.GenerateTextToImageResponse, error) {
	// client.logger.Debugf("metadata for request %+v", )
	switch providerName := strings.ToLower(request.ProviderName); providerName {
	case "stabilityai":
		return client.generateImageOpenAI(c, request)
	default:
		return client.generateImageOpenAI(c, request)
	}
}

func (client *integrationServiceClient) generateImageOpenAI(c context.Context, request *clients_pogos.RequestData[string]) (*integration_api.GenerateTextToImageResponse, error) {
	_metadata := clients_pogos.GenerateAuditInfo(request)
	body := &integration_api.GenerateTextToImageRequest{
		Version: request.Version,
		Model:   request.ProviderModelName,
		Prompt:  request.GlobalPrompt,
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		AuditInfo:       _metadata,
	}
	res, err := client.openAiClient.GenerateTextToImage(c, body)
	if err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	} else {
		return res, nil
	}
}

/*
text base prompt resolutions usually calls generate api in integration service
the apis may be the same but make it more concret so modification becomes easy
*/
func (client *integrationServiceClient) promptAnthropic(c context.Context, request *clients_pogos.RequestData[string]) (*integration_api.GenerateResponse, error) {
	body := &integration_api.GenerateRequest{
		Version:      request.Version,
		SystemPrompt: request.SystemPrompt,
		Prompt:       request.GlobalPrompt,
		Model:        request.ProviderModelName,
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		AuditInfo:       clients_pogos.GenerateAuditInfo(request),
	}
	if res, err := client.anthropicClient.Generate(c, body); err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	} else {
		return res, nil
	}
}

func (client *integrationServiceClient) promptOpenAI(c context.Context, request *clients_pogos.RequestData[string]) (*integration_api.GenerateResponse, error) {
	_metadata := clients_pogos.GenerateAuditInfo(request)
	body := &integration_api.GenerateRequest{
		Version:      request.Version,
		Model:        request.ProviderModelName,
		Prompt:       request.GlobalPrompt,
		SystemPrompt: request.SystemPrompt,
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		AuditInfo:       _metadata,
	}
	client.logger.Errorf("Making call to client %+v", body)
	res, err := client.openAiClient.Generate(c, body)
	if err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	} else {
		return res, nil
	}
}

func (client *integrationServiceClient) promptCohere(c context.Context, request *clients_pogos.RequestData[string]) (*integration_api.GenerateResponse, error) {
	body := &integration_api.GenerateRequest{
		Version:      request.Version,
		Model:        request.ProviderModelName,
		Prompt:       request.GlobalPrompt,
		SystemPrompt: request.SystemPrompt,
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		AuditInfo:       clients_pogos.GenerateAuditInfo(request),
	}
	if res, err := client.cohereClient.Generate(c, body); err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	} else {
		return res, nil
	}
}

func (client *integrationServiceClient) promptGoogle(c context.Context, request *clients_pogos.RequestData[string]) (*integration_api.GenerateResponse, error) {
	body := &integration_api.GenerateRequest{
		Version:      request.Version,
		Model:        request.ProviderModelName,
		Prompt:       request.GlobalPrompt,
		SystemPrompt: request.SystemPrompt,
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		AuditInfo:       clients_pogos.GenerateAuditInfo(request),
	}
	res, err := client.googleClient.Generate(c, body)
	if err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	}
	return res, nil
}

func (client *integrationServiceClient) promptReplicate(c context.Context, request *clients_pogos.RequestData[string]) (*integration_api.GenerateResponse, error) {
	body := &integration_api.GenerateRequest{
		Version:      request.Version,
		Model:        request.ProviderModelName,
		Prompt:       request.GlobalPrompt,
		SystemPrompt: request.SystemPrompt,
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		AuditInfo:       clients_pogos.GenerateAuditInfo(request),
	}
	res, err := client.replicateClient.Generate(c, body)
	if err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	}
	return res, nil
}

/*
All the conversation request
*/

func (client *integrationServiceClient) converseWithCohere(c context.Context, request *clients_pogos.RequestData[[]*clients_pogos.Interaction]) (*integration_api.ChatResponse, error) {
	// currentPrompt := request.Conversations[len(request.Conversations)-1]
	body := &integration_api.ChatRequest{
		Model: request.ProviderModelName,
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		Version:         request.Version,
		Conversations:   clients_pogos.ToConversaction(request.GlobalPrompt),
		AuditInfo:       clients_pogos.GenerateAuditInfo(request),
	}

	if res, err := client.cohereClient.Chat(c, body); err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	} else {
		return res, nil
	}
}

func (client *integrationServiceClient) converseWithAnthropic(c context.Context, request *clients_pogos.RequestData[[]*clients_pogos.Interaction]) (*integration_api.ChatResponse, error) {

	body := &integration_api.ChatRequest{
		Model: request.ProviderModelName,
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		Version:         request.Version,
		Conversations:   clients_pogos.ToConversaction(request.GlobalPrompt),
		AuditInfo:       clients_pogos.GenerateAuditInfo(request),
	}

	if res, err := client.anthropicClient.Chat(c, body); err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	} else {
		return res, nil
	}
}

func (client *integrationServiceClient) converseWithOpenAI(c context.Context, request *clients_pogos.RequestData[[]*clients_pogos.Interaction]) (*integration_api.ChatResponse, error) {

	body := &integration_api.ChatRequest{
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		Model:           request.ProviderModelName,
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		Version:         request.Version,
		Conversations:   clients_pogos.ToConversaction(request.GlobalPrompt),
		AuditInfo:       clients_pogos.GenerateAuditInfo(request),
	}

	if res, err := client.openAiClient.Chat(c, body); err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	} else {
		return res, nil
	}
}

func (client *integrationServiceClient) converseWithGoogle(c context.Context, request *clients_pogos.RequestData[[]*clients_pogos.Interaction]) (*integration_api.ChatResponse, error) {

	body := &integration_api.ChatRequest{
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		Model:           request.ProviderModelName,
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		Version:         request.Version,
		Conversations:   clients_pogos.ToConversaction(request.GlobalPrompt),
		AuditInfo:       clients_pogos.GenerateAuditInfo(request),
	}

	if res, err := client.googleClient.Chat(c, body); err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	} else {
		return res, nil
	}
}

func (client *integrationServiceClient) converseWithReplicate(c context.Context, request *clients_pogos.RequestData[[]*clients_pogos.Interaction]) (*integration_api.ChatResponse, error) {

	body := &integration_api.ChatRequest{
		Credential: &integration_api.Credential{
			Id:    request.Credential.Id,
			Value: request.Credential.Key,
		},
		ModelParameters: clients_pogos.GenerateModelParameter(request.ProviderModelParameters),
		Model:           request.ProviderModelName,
		Version:         request.Version,
		Conversations:   clients_pogos.ToConversaction(request.GlobalPrompt),
		AuditInfo:       clients_pogos.GenerateAuditInfo(request),
	}

	if res, err := client.replicateClient.Chat(c, body); err != nil {
		client.logger.Error("Error occured %v", err)
		return nil, err
	} else {
		return res, nil
	}
}

func (client *integrationServiceClient) WelcomeEmail(c context.Context, userId uint64, name, email string) (*integration_api.WelcomeEmailResponse, error) {
	client.logger.Debugf("sending welcome email from integration client")
	res, err := client.sendgridClient.WelcomeEmail(c, &integration_api.WelcomeEmailRequest{
		UserId: userId,
		To: &integration_api.Contact{
			Name:  name,
			Email: email,
		},
	})
	if err != nil {
		client.logger.Errorf("unable to send welcome email error %v", err)
		return nil, err
	}
	return res, nil

}

func (client *integrationServiceClient) GetAuditLog(c context.Context, organizationId, projectId, auditId uint64) (*integration_api.GetAuditLogResponse, error) {
	client.logger.Debugf("Calling to get audit log with org and project")
	res, err := client.auditLoggingClient.GetAuditLog(c, &integration_api.GetAuditLogRequest{
		OrganizationId: organizationId,
		ProjectId:      projectId,
		Id:             auditId,
	})
	if err != nil {
		client.logger.Errorf("error while getting audit log error %v", err)
		return nil, err
	}
	return res, nil
}
func (client *integrationServiceClient) GetAllAuditLog(c context.Context, organizationId, projectId uint64, criterias []*integration_api.Criteria, paginate *integration_api.Paginate) (*integration_api.GetAllAuditLogResponse, error) {
	client.logger.Debugf("Calling to get audit log with org and project")
	res, err := client.auditLoggingClient.GetAllAuditLog(c, &integration_api.GetAllAuditLogRequest{
		OrganizationId: organizationId,
		ProjectId:      projectId,
		Criterias:      criterias,
		Paginate:       paginate,
	})
	if err != nil {
		client.logger.Errorf("error while getting audit log error %v", err)
		return nil, err
	}
	return res, nil
}

func (client *integrationServiceClient) ResetPasswordEmail(c context.Context, userId uint64, name, email, resetPasswordLink string) (*integration_api.ResetPasswordEmailResponse, error) {
	client.logger.Debugf("sending reset password email from integration client")
	res, err := client.sendgridClient.ResetPasswordEmail(c, &integration_api.ResetPasswordEmailRequest{
		UserId: userId,
		To: &integration_api.Contact{
			Name:  name,
			Email: email,
		},
		ResetPasswordLink: resetPasswordLink,
	})
	if err != nil {
		client.logger.Errorf("unable to send reset password link error %v", err)
		return nil, err
	}
	return res, nil
}

func (client *integrationServiceClient) InviteMemberEmail(c context.Context, userId uint64, name, email, organizationName, projectName, inviterName string) (*integration_api.InviteMemeberEmailResponse, error) {
	client.logger.Debugf("sending invite member email from integration client")
	res, err := client.sendgridClient.InviteMemberEmail(c, &integration_api.InviteMemeberEmailRequest{
		UserId: userId,
		To: &integration_api.Contact{
			Name:  name,
			Email: email,
		},
		OrganizationName: organizationName,
		ProjectName:      projectName,
		InviterName:      inviterName,
	})
	if err != nil {
		client.logger.Errorf("unable to send invite member email error %v", err)
		return nil, err
	}
	return res, nil
}
