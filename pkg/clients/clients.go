package internal_clients

import (
	"context"

	_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type ProviderServiceClient interface {
	GetAllProviders(c context.Context) (*_api.GetAllProviderResponse, error)
}

type IntegrationServiceClient interface {
	WelcomeEmail(c context.Context, userId uint64, name, email string) (*_api.WelcomeEmailResponse, error)
	ResetPasswordEmail(c context.Context, userId uint64, name, email, resetPasswordLink string) (*_api.ResetPasswordEmailResponse, error)
	InviteMemberEmail(c context.Context, userId uint64, name, email, organizationName, projectName, inviterName string) (*_api.InviteMemeberEmailResponse, error)
	GetAuditLog(c context.Context, organizationId, projectId uint64, criterias []*_api.Criteria, paginate *_api.Paginate) (*_api.GetAuditLogResponse, error)
}

type EndpointServiceClient interface {
	GetAllEndpoint(c context.Context, projectId, organizationId uint64, criterias []*_api.Criteria, paginate *_api.Paginate) (*_api.GetAllEndpointResponse, error)
	GetEndpoint(c context.Context, endpointId uint64, projectId, organizationId uint64) (*_api.GetEndpointResponse, error)
	CreateEndpoint(c context.Context, endpointRequest *_api.CreateEndpointRequest, projectId, organizationId, userId uint64) (*_api.EndpointProviderModelResponse, error)
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
