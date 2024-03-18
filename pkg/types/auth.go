package types

import (
	"context"
)

var (
	CTX_              string = "__auth"
	AUTHORIZATION_KEY        = "Authorization"
	AUTH_KEY                 = "x-auth-id"
	PROJECT_KEY              = "x-project-id"
	SERVICE_SCOPE_KEY        = "x-internal-service-key"
	PROJECT_SCOPE_KEY        = "x-api-key"
	ORG_SCOPE_KEY            = "x-org-key"
)

type Authenticator interface {
	Authorize(ctx context.Context, authToken string, userId uint64) (Principle, error)
	AuthPrinciple(ctx context.Context, userId uint64) (Principle, error)
}

type ClaimAuthenticator[T SimplePrinciple] interface {
	Claim(ctx context.Context, claimToken string) (*PlainClaimPrinciple[T], error)
}

type PlainClaimPrinciple[T SimplePrinciple] struct {
	Info T `json:"info"`
}

/*
An simple principle that can be used for passing and recieving the data
*/
type SimplePrinciple interface {
	GetUserId() *uint64
	// later will support the user can be part of multiple org
	GetCurrentOrganizationId() *uint64
	// current project context
	GetCurrentProjectId() *uint64
	// has an user
	HasUser() bool
	// has an org
	HasOrganization() bool
	// has an project
	HasProject() bool

	IsAuthenticated() bool
}

/*
 A large priciple
*/

type Principle interface {
	SimplePrinciple
	GetAuthToken() *AuthToken
	GetOrganizationRole() *OrganizaitonRole
	GetUserInfo() *UserInfo
	GetProjectRoles() []*ProjectRole
	GetCurrentProjectRole() *ProjectRole

	// return a concrete type
	PlainAuthPrinciple() PlainAuthPrinciple
	SwitchProject(projectId uint64) error
}
