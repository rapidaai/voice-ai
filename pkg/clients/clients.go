package internal_clients

import (
	"context"

	clients_pogos "github.com/lexatic/web-backend/pkg/clients/pogos"
	"github.com/lexatic/web-backend/pkg/types"
	_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type ProviderServiceClient interface {
	GetAllProviders(c context.Context) (*_api.GetAllProviderResponse, error)
}

type IntegrationServiceClient interface {
	WelcomeEmail(c context.Context, userId uint64, name, email string) (*_api.WelcomeEmailResponse, error)
	ResetPasswordEmail(c context.Context, userId uint64, name, email, resetPasswordLink string) (*_api.ResetPasswordEmailResponse, error)
	InviteMemberEmail(c context.Context, userId uint64, name, email, organizationName, projectName, inviterName string) (*_api.InviteMemeberEmailResponse, error)
	GetAuditLog(c context.Context, organizationId, projectId, auditId uint64) (*_api.GetAuditLogResponse, error)
	Converse(c context.Context, request *clients_pogos.RequestData[[]*clients_pogos.Interaction]) (*_api.ChatResponse, error)
	Prompt(c context.Context, request *clients_pogos.RequestData[string]) (*_api.GenerateResponse, error)
	GenerateTextToImage(c context.Context, request *clients_pogos.RequestData[string]) (*_api.GenerateTextToImageResponse, error)
	GetAllAuditLog(c context.Context, organizationId, projectId uint64, criterias []*_api.Criteria, paginate *_api.Paginate) (*_api.GetAllAuditLogResponse, error)
}

type EndpointServiceClient interface {
	GetAllEndpoint(c context.Context, projectId, organizationId uint64, criterias []*_api.Criteria, paginate *_api.Paginate) (*_api.GetAllEndpointResponse, error)
	GetAllEndpointProviderModel(c context.Context, endpointId, projectId, organizationId uint64, criterias []*_api.Criteria, paginate *_api.Paginate) (*_api.GetAllEndpointProviderModelResponse, error)
	GetEndpoint(c context.Context, endpointId uint64, projectId, organizationId uint64) (*_api.GetEndpointResponse, error)
	UpdateEndpointVersion(c context.Context, endpointId, endpointProviderModelId, updatedBy, projectId, organizationId uint64) (*_api.UpdateEndpointVersionResponse, error)
	CreateEndpoint(c context.Context, endpointRequest *_api.CreateEndpointRequest, projectId, organizationId, userId uint64) (*_api.CreateEndpointProviderModelResponse, error)
	CreateEndpointFromTestcase(c context.Context, iRequest *_api.CreateEndpointFromTestcaseRequest, principle *types.PlainAuthPrinciple) (*_api.CreateEndpointProviderModelResponse, error)
	CreateEndpointProviderModel(c context.Context, endpointRequest *_api.CreateEndpointRequest, projectId, organizationId, userId uint64) (*_api.CreateEndpointProviderModelResponse, error)
}

type WebhookServiceClient interface {
	CreateWebhook(c context.Context,
		url, description string, eventType []string, maxRetryCount uint32,
		userId, projectId, organizationId uint64,
	) (*_api.CreateWebhookResponse, error)
	DisableWebhook(ctx context.Context, id, projectId, organizationId uint64) (*_api.DisableWebhookResponse, error)
	DeleteWebhook(ctx context.Context, id, projectId, organizationId uint64) (*_api.DeleteWebhookResponse, error)
	GetAllWebhook(ctx context.Context, organizationId, projectId uint64, criterias []*_api.Criteria, paginate *_api.Paginate) (*_api.GetAllWebhookResponse, error)
	GetWebhook(c context.Context, webhookId uint64, projectId, organizationId uint64) (*_api.GetWebhookResponse, error)
}

type TestingServiceClient interface {
	GetTestSuite(c context.Context, testsuiteId uint64) (*_api.GetTestSuiteResponse, error)
}
