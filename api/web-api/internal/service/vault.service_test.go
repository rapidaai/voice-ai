package internal_service

import (
	"context"
	"testing"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type normalizedVaultService struct {
	createdAuth    *types.Authentication
	createdProject types.ProjectContext
}

func (s *normalizedVaultService) Create(_ context.Context, auth *types.Authentication, projectContext types.ProjectContext, _ string, _ string, _ map[string]interface{}) (*internal_entity.Vault, error) {
	s.createdAuth = auth
	s.createdProject = projectContext
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) Get(context.Context, types.ProjectContext, uint64) (*internal_entity.Vault, error) {
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) GetProviderCredential(context.Context, uint64, string) (*internal_entity.Vault, error) {
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) Delete(context.Context, *types.Authentication, types.ProjectContext, uint64) (*internal_entity.Vault, error) {
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) GetAllOrganizationCredential(context.Context, types.ProjectContext, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.Vault, error) {
	return 0, nil, nil
}

var _ VaultService = (*normalizedVaultService)(nil)

func TestVaultServiceContractUsesNormalizedIdentityInputs(t *testing.T) {
	service := &normalizedVaultService{}
	projectContext := types.ProjectContext{OrganizationID: 11, ProjectID: 22}
	actor := types.ActorIdentity{Type: types.ActorTypeUser, ID: 7}
	auth := &types.Authentication{AuthType: types.AuthTypeUser, ActorValue: &actor}

	if _, err := service.Create(context.Background(), auth, projectContext, "provider", "name", nil); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if service.createdAuth != auth || service.createdProject != projectContext {
		t.Fatalf("Create() identity inputs = auth:%p project:%+v", service.createdAuth, service.createdProject)
	}
}
