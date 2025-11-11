package types

import (
	"context"

	"github.com/gin-gonic/gin"
)

type CTX_KEY string

var (
	CTX_              CTX_KEY = "__auth"
	AUTHORIZATION_KEY         = "authorization"
	AUTH_KEY                  = "x-auth-id"
	PROJECT_KEY               = "x-project-id"
	SERVICE_SCOPE_KEY         = "x-internal-service-key"

	//
	PROJECT_SCOPE_KEY = "x-api-key"
	// later we will check the prefix and drop the request, this will not overload the server with random request
	// another way to find length and pattern of our generated key, validate first if the given key in the same format
	// only needed for scale
	ORG_SCOPE_KEY      = "x-org-key"
	ORG_KEY_PREFIX     = "rpd-org-"
	PROJECT_KEY_PREFIX = "rpd-prj-"
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
	//
	IsAuthenticated() bool
	//
	GetCurrentToken() string

	Type() string
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

	//
	PlainAuthPrinciple() PlainAuthPrinciple
	SwitchProject(projectId uint64) error
	GetFeaturePermission() []*FeaturePermission
}

func GetAuthPrincipleGPRC(ctx context.Context) (Principle, bool) {
	ath := ctx.Value(CTX_)
	switch md := ath.(type) {
	case Principle:
		return md, true
	default:
		return nil, false
	}
}

func GetScopePrincipleGRPC[T SimplePrinciple](ctx context.Context) (SimplePrinciple, bool) {
	ath := ctx.Value(CTX_)
	switch md := ath.(type) {
	case *PlainClaimPrinciple[T]:
		return md.Info, md.Info.IsAuthenticated()
	case Principle:
		return md, md.IsAuthenticated()
	default:
		return nil, false
	}
}

func GetSimplePrincipleGRPC(ctx context.Context) (SimplePrinciple, bool) {
	ath := ctx.Value(CTX_)
	switch md := ath.(type) {
	case *PlainClaimPrinciple[*ProjectScope]:
		return md.Info, md.Info.IsAuthenticated()
	case *PlainClaimPrinciple[*ServiceScope]:
		return md.Info, md.Info.IsAuthenticated()
	case *PlainClaimPrinciple[*OrganizationScope]:
		return md.Info, md.Info.IsAuthenticated()
	case Principle:
		return md, md.IsAuthenticated()
	default:
		return nil, false
	}
}

// get auth principle for gin
func GetAuthPrinciple(ctx *gin.Context) (SimplePrinciple, bool) {
	ath, _ := ctx.Get(string(CTX_))
	switch md := ath.(type) {
	case *PlainClaimPrinciple[*ProjectScope]:
		return md.Info, md.Info.IsAuthenticated()
	case *PlainClaimPrinciple[*ServiceScope]:
		return md.Info, md.Info.IsAuthenticated()
	case *PlainClaimPrinciple[*OrganizationScope]:
		return md.Info, md.Info.IsAuthenticated()
	case Principle:
		return md, md.IsAuthenticated()

	default:
		return nil, false
	}
}
