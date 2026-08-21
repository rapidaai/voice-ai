package internal_service

import (
	"context"
	"testing"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type normalizedVaultService struct {
	createdUserID  uint64
	createdProject types.ProjectContext
}

func (s *normalizedVaultService) Create(_ context.Context, userID uint64, projectContext types.ProjectContext, _ string, _ string, _ map[string]interface{}) (*internal_entity.Vault, error) {
	s.createdUserID = userID
	s.createdProject = projectContext
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) Get(context.Context, types.ProjectContext, uint64) (*internal_entity.Vault, error) {
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) GetProviderCredential(context.Context, uint64, string) (*internal_entity.Vault, error) {
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) Delete(context.Context, uint64, types.ProjectContext, uint64) (*internal_entity.Vault, error) {
	return &internal_entity.Vault{}, nil
}

func (s *normalizedVaultService) GetAllOrganizationCredential(context.Context, types.ProjectContext, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.Vault, error) {
	return 0, nil, nil
}

var _ VaultService = (*normalizedVaultService)(nil)

func TestVaultServiceContractUsesNormalizedIdentityInputs(t *testing.T) {
	service := &normalizedVaultService{}
	projectContext := types.ProjectContext{OrganizationID: 11, ProjectID: 22}

	if _, err := service.Create(context.Background(), 33, projectContext, "provider", "name", nil); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if service.createdUserID != 33 || service.createdProject != projectContext {
		t.Fatalf("Create() identity inputs = user:%d project:%+v", service.createdUserID, service.createdProject)
	}
}
