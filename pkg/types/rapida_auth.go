// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
)

type CTX_KEY string

var (
	CTX_              CTX_KEY = "__auth"
	AUTHORIZATION_KEY         = "authorization"
	AUTH_KEY                  = "x-auth-id"
	PROJECT_KEY               = "x-project-id"
	SERVICE_SCOPE_KEY         = "x-internal-service-key"

	PROJECT_SCOPE_KEY  = "x-api-key"
	ORG_SCOPE_KEY      = "x-org-key"
	ORG_KEY_PREFIX     = "rpd-org-"
	PROJECT_KEY_PREFIX = "rpd-prj-"
)

type AuthType string

const (
	AuthTypeUser    AuthType = "user"
	AuthTypeService AuthType = "service"
	AuthTypeSystem  AuthType = "system"
	AuthTypeProject AuthType = "project"
	AuthTypeOrg     AuthType = "organization"
)

func (a AuthType) String() string {
	return string(a)
}

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

type AuthenticationPrinciple interface {
	IsAuthenticated() bool
	GetCurrentToken() string
	Type() AuthType
}

var (
	ErrUnauthenticated                    = errors.New("authentication required")
	ErrAuthenticationScopeNotAllowed      = errors.New("authentication scope is not allowed")
	ErrActorUnavailable                   = errors.New("actor identity is unavailable")
	ErrUserContextUnavailable             = errors.New("user context is unavailable")
	ErrOrganizationContextUnavailable     = errors.New("organization context is unavailable")
	ErrProjectContextUnavailable          = errors.New("project context is unavailable")
	ErrInvalidServiceAssertion            = errors.New("service assertion is invalid")
	ErrInvalidDelegatedIdentity           = errors.New("delegated identity is invalid")
	ErrUnsupportedDelegatedAuthentication = errors.New("delegated authentication type is unsupported")
	ErrAuthenticationContextMismatch      = errors.New("authentication context does not match actor")
	ErrServiceNameUnavailable             = errors.New("service name is unavailable")
	ErrServiceActorUnavailable            = errors.New("service actor identity is unavailable")
	ErrServiceSecretUnavailable           = errors.New("service secret is unavailable")
)

type UserContext struct {
	UserID uint64
}

type OrganizationContext struct {
	OrganizationID uint64
}

type ProjectContext struct {
	OrganizationID uint64
	ProjectID      uint64
}

type Authentication struct {
	AuthType   AuthType
	ActorValue *ActorIdentity

	UserValue         *UserContext
	OrganizationValue *OrganizationContext
	ProjectValue      *ProjectContext
}

func (auth *Authentication) IsAuthenticated() bool {
	if auth == nil || auth.ActorValue == nil || auth.ActorValue.Validate() != nil || auth.ActorValue.Type != ActorType(auth.AuthType) {
		return false
	}
	switch auth.AuthType {
	case AuthTypeUser:
		user, userErr := auth.UserContext()
		_, organizationErr := auth.OrganizationContext()
		return userErr == nil && organizationErr == nil && user.UserID == auth.ActorValue.ID
	case AuthTypeProject:
		_, projectErr := auth.ProjectContext()
		return projectErr == nil
	case AuthTypeOrg, AuthTypeService:
		_, organizationErr := auth.OrganizationContext()
		return organizationErr == nil
	case AuthTypeSystem:
		return true
	default:
		return false
	}
}
func (auth *Authentication) GetCurrentToken() string { return "" }
func (auth *Authentication) Type() AuthType          { return auth.AuthType }
func (auth *Authentication) Scope(allowed ...AuthType) (*Authentication, error) {
	if !auth.IsAuthenticated() {
		return nil, ErrUnauthenticated
	}
	for _, authType := range allowed {
		if authType == auth.AuthType {
			return auth, nil
		}
	}
	return nil, ErrAuthenticationScopeNotAllowed
}
func (auth *Authentication) Actor() ActorIdentity {
	if auth == nil || auth.ActorValue == nil {
		return ActorIdentity{}
	}
	return *auth.ActorValue
}
func (auth *Authentication) UserContext() (UserContext, error) {
	if auth == nil || auth.UserValue == nil || auth.UserValue.UserID == 0 {
		return UserContext{}, ErrUserContextUnavailable
	}
	return *auth.UserValue, nil
}
func (auth *Authentication) OrganizationContext() (OrganizationContext, error) {
	if auth == nil || auth.OrganizationValue == nil || auth.OrganizationValue.OrganizationID == 0 {
		return OrganizationContext{}, ErrOrganizationContextUnavailable
	}
	return *auth.OrganizationValue, nil
}
func (auth *Authentication) ProjectContext() (ProjectContext, error) {
	if auth == nil || auth.ProjectValue == nil || auth.ProjectValue.OrganizationID == 0 || auth.ProjectValue.ProjectID == 0 {
		return ProjectContext{}, ErrProjectContextUnavailable
	}
	return *auth.ProjectValue, nil
}

func Authorize(ctx context.Context) (*Authentication, error) {
	auth, ok := ctx.Value(CTX_).(*Authentication)
	if !ok || auth == nil || !auth.IsAuthenticated() {
		return nil, ErrUnauthenticated
	}
	return auth, nil
}

// AuthorizeUser returns valid user authentication without requiring organization context.
func AuthorizeUser(ctx context.Context) (*Authentication, error) {
	auth, ok := ctx.Value(CTX_).(*Authentication)
	if !ok || auth == nil || auth.AuthType != AuthTypeUser {
		return nil, ErrUnauthenticated
	}
	userContext, err := auth.UserContext()
	if err != nil {
		return nil, ErrUnauthenticated
	}
	actor := auth.Actor()
	if actor.Validate() != nil || actor.Type != ActorTypeUser || actor.ID != userContext.UserID {
		return nil, ErrUnauthenticated
	}
	return auth, nil
}

// SimplePrinciple is retained for source compatibility.
// Deprecated: use Authentication for request authentication.
type SimplePrinciple = AuthenticationPrinciple

type UserIdentityProvider interface {
	UserIdentity() (uint64, bool)
}

type OrganizationContextProvider interface {
	OrganizationContext() (uint64, bool)
}

type ProjectContextProvider interface {
	ProjectContext() (ProjectContext, bool)
}

type DelegatedContextProvider interface {
	DelegatedContext() (DelegatedContext, bool)
}

/*
 A large principle
*/

// Principle is retained for source compatibility with legacy authenticators.
// Deprecated: use Authentication for request authentication.
type Principle interface {
	AuthenticationPrinciple
	UserIdentityProvider
	OrganizationContextProvider
	GetAuthToken() *AuthToken
	GetOrganizationRole() *OrganizaitonRole
	GetUserInfo() *UserInfo
	GetProjectRoles() []*ProjectRole
	GetCurrentProjectRole() *ProjectRole

	PlainAuthPrinciple() PlainAuthPrinciple
	SwitchProject(projectId uint64) error
	GetFeaturePermission() []*FeaturePermission
}

// GetAuthPrincipleGPRC reads a legacy principle from context.
// Deprecated: use Authorize.
func GetAuthPrincipleGPRC(ctx context.Context) (Principle, bool) {
	ath := ctx.Value(CTX_)
	switch md := ath.(type) {
	case Principle:
		return md, true
	default:
		return nil, false
	}
}

// GetScopePrincipleGRPC reads a legacy scoped principle from context.
// Deprecated: use Authorize and Authentication.Scope.
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
