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

type Authentication interface {
	AuthenticationPrinciple
	Scope(allowed ...AuthType) (AuthenticationPrinciple, error)
}

var (
	ErrUnauthenticated               = errors.New("authentication required")
	ErrAuthenticationScopeNotAllowed = errors.New("authentication scope is not allowed")
)

func Authorize(ctx context.Context) (Authentication, error) {
	auth, ok := ctx.Value(CTX_).(Authentication)
	if !ok || auth == nil || !auth.IsAuthenticated() {
		return nil, ErrUnauthenticated
	}
	return auth, nil
}

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
