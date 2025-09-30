package internal_service

import (
	"context"

	internal_entity "github.com/lexatic/web-backend/api/web-api/internal/entity"
	"github.com/lexatic/web-backend/pkg/types"
	type_enums "github.com/lexatic/web-backend/pkg/types/enums"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type UserService interface {
	Authenticate(ctx context.Context, email string, password string) (types.Principle, error)
	AuthPrinciple(ctx context.Context, userId uint64) (types.Principle, error)

	Get(ctx context.Context, email string) (*internal_entity.UserAuth, error)
	GetUser(ctx context.Context, userId uint64) (*internal_entity.UserAuth, error)
	UpdateUser(ctx context.Context, auth types.Principle, userId uint64, name *string) (*internal_entity.UserAuth, error)
	UpdatePassword(ctx context.Context, userId uint64, password string) (*internal_entity.UserAuth, error)
	GetToken(ctx context.Context, tokenType string, token string) (*internal_entity.UserAuthToken, error)
	Create(ctx context.Context, name string, email string, password string, status type_enums.RecordState, source *string) (types.Principle, error)
	CreatePasswordToken(ctx context.Context, userId uint64) (*internal_entity.UserAuthToken, error)
	//
	CreateOrganizationRole(ctx context.Context, auth types.Principle, role string, userId uint64, orgnizationId uint64, status type_enums.RecordState) (*internal_entity.UserOrganizationRole, error)
	CreateProjectRole(ctx context.Context, auth types.Principle, userId uint64, role string, projectId uint64, status type_enums.RecordState) (*internal_entity.UserProjectRole, error)

	//
	ActivateAllProjectRoles(ctx context.Context, userId uint64) error
	ActivateAllOrganizationRole(ctx context.Context, userId uint64) error
	GetAllOrganizationMember(ctx context.Context, organizationId uint64, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_entity.UserOrganizationRole, error)

	GetProjectRole(ctx context.Context, userId uint64, projectId uint64) (*internal_entity.UserProjectRole, error)
	GetOrganizationRole(ctx context.Context, userId uint64) (*internal_entity.UserOrganizationRole, error)
	Activate(ctx context.Context, Id uint64, name string, source *string) (types.Principle, error)

	CreateSocial(ctx context.Context, userId uint64, id string, token string, source string, verified bool) (*internal_entity.UserSocial, error)
	GetSocial(ctx context.Context, userId uint64) (*internal_entity.UserSocial, error)

	GetAllActiveProjectMember(ctx context.Context, projectId uint64) (*[]internal_entity.UserProjectRole, error)
	GetAllUserRolesForOrg(ctx context.Context, organizationId uint64) ([]*internal_entity.UserOrganizationRole, error)
	GetProjectRolesForUsers(ctx context.Context, pIds []uint64, uIds []uint64) ([]*internal_entity.UserProjectRole, error)

	GetAllUserFeaturePermission(ctx context.Context, userId uint64) ([]*internal_entity.UserFeaturePermission, error)
	EnableAllDefaultUserFeaturePermission(ctx context.Context, userId uint64) ([]*internal_entity.UserFeaturePermission, error)
}
