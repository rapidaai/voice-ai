package internal_service

import (
	"context"
	"testing"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/protos"
)

type normalizedVaultService struct {
	createdAuth authentication.Principal
}

func (s *normalizedVaultService) Create(_ context.Context, auth authentication.Principal, _ string, _ string, _ map[string]interface{}) (*internal_entity.Vault, error) {
	s.createdAuth = auth
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) Get(context.Context, authentication.Tenant, uint64) (*internal_entity.Vault, error) {
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) GetProviderCredential(context.Context, uint64, string) (*internal_entity.Vault, error) {
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) Delete(context.Context, authentication.Principal, uint64) (*internal_entity.Vault, error) {
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) GetAllOrganizationCredential(context.Context, authentication.Principal, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.Vault, error) {
	return 0, nil, nil
}

var _ VaultService = (*normalizedVaultService)(nil)

func TestVaultServiceContractUsesNormalizedIdentityInputs(t *testing.T) {
	service := &normalizedVaultService{}
	auth, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7},
		authentication.Tenant{OrganizationID: 11, ProjectID: 22},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "vault-contract-test"}}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.Create(context.Background(), auth, "provider", "name", nil); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if service.createdAuth.Subject() != auth.Subject() || service.createdAuth.Tenant() != auth.Tenant() {
		t.Fatalf("Create() identity input = %+v", service.createdAuth)
	}
}
