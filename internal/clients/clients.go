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
}
