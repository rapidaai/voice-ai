package internal_service

import (
	"context"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
	web_api "github.com/rapidaai/protos"
)

type VaultService interface {
	Create(ctx context.Context,
		auth authentication.Principal,
		provider string,
		name string, credential map[string]interface{}) (*internal_entity.Vault, error)
	Get(ctx context.Context, tenant authentication.Tenant, vltId uint64) (*internal_entity.Vault, error)
	GetProviderCredential(ctx context.Context, organizationID uint64, provider string) (*internal_entity.Vault, error)
	Delete(ctx context.Context, auth authentication.Principal, vaultId uint64) (*internal_entity.Vault, error)
	GetAllOrganizationCredential(ctx context.Context, auth authentication.Principal, criteria []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_entity.Vault, error)
}
