package internal_services

import (
	"context"

	internal_gorm "github.com/lexatic/web-backend/internal/gorm"
	"github.com/lexatic/web-backend/pkg/types"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type UserService interface {
	Authenticate(ctx context.Context, email string, password string) (types.Principle, error)
	AuthPrinciple(ctx context.Context, userId uint64) (types.Principle, error)

	Get(ctx context.Context, email string) (*internal_gorm.UserAuth, error)
	GetUser(ctx context.Context, userId uint64) (*internal_gorm.UserAuth, error)
	UpdateUser(ctx context.Context, auth types.Principle, userId uint64, name *string) (*internal_gorm.UserAuth, error)
	UpdatePassword(ctx context.Context, userId uint64, password string) (*internal_gorm.UserAuth, error)
	GetToken(ctx context.Context, tokenType string, token string) (*internal_gorm.UserAuthToken, error)
	Create(ctx context.Context, name string, email string, password string, staus string, source *string) (types.Principle, error)
	CreatePasswordToken(ctx context.Context, userId uint64) (*internal_gorm.UserAuthToken, error)
	//
	CreateOrganizationRole(ctx context.Context, auth types.Principle, role string, userId uint64, orgnizationId uint64, status string) (*internal_gorm.UserOrganizationRole, error)
	CreateProjectRole(ctx context.Context, auth types.Principle, userId uint64, role string, projectId uint64, status string) (*internal_gorm.UserProjectRole, error)

	//
	ActivateAllProjectRoles(ctx context.Context, userId uint64) error
	ActivateAllOrganizationRole(ctx context.Context, userId uint64) error
	GetAllOrganizationMember(ctx context.Context, organizationId uint64, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_gorm.UserOrganizationRole, error)

	GetProjectRole(ctx context.Context, userId uint64, projectId uint64) (*internal_gorm.UserProjectRole, error)
	GetOrganizationRole(ctx context.Context, userId uint64) (*internal_gorm.UserOrganizationRole, error)
	//
	Activate(ctx context.Context, Id uint64, name string, source *string) (types.Principle, error)
	// GetAllUser(ctx context.Context, uIds []uint64, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_gorm.UserAuth, error)

	CreateSocial(ctx context.Context, userId uint64, id string, token string, source string, verified bool) (*internal_gorm.UserSocial, error)
	GetSocial(ctx context.Context, userId uint64) (*internal_gorm.UserSocial, error)
	// GetUsers(ctx context.Context, uIds []uint64, limit uint32, offset uint32) ([]*internal_gorm.UserAuth, error)

	// GetOrInviteUser(ctx context.Context, email string, organizationId uint64) (*internal_gorm.UserAuth, error)
	GetAllActiveProjectMember(ctx context.Context, projectId uint64) (*[]internal_gorm.UserProjectRole, error)
	GetAllUserRolesForOrg(ctx context.Context, organizationId uint64) ([]*internal_gorm.UserOrganizationRole, error)
	GetProjectRolesForUsers(ctx context.Context, pIds []uint64, uIds []uint64) ([]*internal_gorm.UserProjectRole, error)
	// GetAllProjectMembers(ctx context.Context, projectId uint64) (*[]internal_gorm.UserProjectRole, error)
}

type OrganizationService interface {
	Create(ctx context.Context, auth types.Principle, name string, size string, industry string) (*internal_gorm.Organization, error)
	Get(ctx context.Context, organizationId uint64) (*internal_gorm.Organization, error)
	Update(ctx context.Context, auth types.Principle, organizationId uint64, name *string, industry *string, email *string) (*internal_gorm.Organization, error)
}

type ProjectService interface {
	Create(ctx context.Context, auth types.Principle, organizationId uint64, name string, description string) (*internal_gorm.Project, error)
	Update(ctx context.Context, auth types.Principle, projectId uint64, name *string, description *string) (*internal_gorm.Project, error)
	Get(ctx context.Context, auth types.SimplePrinciple, projectId uint64) (*internal_gorm.Project, error)
	GetAll(ctx context.Context, auth types.SimplePrinciple, organizationId uint64, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_gorm.Project, error)
	Archive(ctx context.Context, auth types.Principle, projectId uint64) (*internal_gorm.Project, error)

	CreateCredential(ctx context.Context, auth types.Principle, name string, projectId, organizationId uint64) (*internal_gorm.ProjectCredential, error)
	ArchiveCredential(ctx context.Context, auth types.Principle, credentialId, projectId, organizationId uint64) (*internal_gorm.ProjectCredential, error)
	GetAllCredential(ctx context.Context, auth types.Principle, projectId, organizationId uint64, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_gorm.ProjectCredential, error)
}

type LeadService interface {
	Create(ctx context.Context, email string) (*internal_gorm.Lead, error)
}

type VaultService interface {
	CreateOrganizationToolCredential(ctx context.Context,
		auth types.Principle,
		toolId uint64,
		name string, credential map[string]interface{}) (*internal_gorm.Vault, error)

	//
	CreateOrganizationProviderCredential(ctx context.Context,
		auth types.Principle,
		providerId uint64,
		name string, credential map[string]interface{}) (*internal_gorm.Vault, error)

	//
	CreateUserToolCredential(ctx context.Context,
		auth types.Principle,
		toolId uint64,
		name string,
		credential map[string]interface{},
	) (*internal_gorm.Vault, error)
	//

	Delete(ctx context.Context, auth types.Principle, vaultId uint64) (*internal_gorm.Vault, error)
	GetAllOrganizationCredential(ctx context.Context, auth types.Principle, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_gorm.Vault, error)
	GetProviderCredential(ctx context.Context, auth types.SimplePrinciple, providerId uint64) (*internal_gorm.Vault, error)
	CreateRapidaProviderCredential(ctx context.Context, organizationId uint64) (*internal_gorm.Vault, error)
}
