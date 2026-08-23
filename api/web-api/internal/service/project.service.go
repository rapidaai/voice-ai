package internal_service

import (
	"context"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	web_api "github.com/rapidaai/protos"
)

type ProjectService interface {
	Create(ctx context.Context, auth *types.Authentication, organizationId uint64, name string, description string) (*internal_entity.Project, error)
	Update(ctx context.Context, auth *types.Authentication, projectId uint64, name *string, description *string) (*internal_entity.Project, error)
	Get(ctx context.Context, auth *types.Authentication, projectId uint64) (*internal_entity.Project, error)
	GetAll(ctx context.Context, auth *types.Authentication, organizationId uint64, criteria []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_entity.Project, error)
	GetAllByOrganization(ctx context.Context, auth *types.Authentication, organizationId uint64, projectIds []uint64) ([]*internal_entity.Project, error)
	Archive(ctx context.Context, auth *types.Authentication, projectId uint64) (*internal_entity.Project, error)

	CreateCredential(ctx context.Context, auth *types.Authentication, name string, projectId, organizationId uint64) (*internal_entity.ProjectCredential, error)
	ArchiveCredential(ctx context.Context, auth *types.Authentication, credentialId, projectId, organizationId uint64) (*internal_entity.ProjectCredential, error)
	GetAllCredential(ctx context.Context, auth *types.Authentication, projectId, organizationId uint64, criteria []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_entity.ProjectCredential, error)
}
