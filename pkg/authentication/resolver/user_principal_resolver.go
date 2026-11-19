// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package resolver

import (
	"context"
	"math"
	"strings"

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/validator"
)

// UserAuthenticationService verifies one User credential and returns its canonical aggregate.
type UserAuthenticationService interface {
	ResolveUserCredential(ctx context.Context, token string, userID uint64, selectedOrganizationID uint64, selectedProjectID uint64) (authentication.Authentication, error)
}

// UserCredential contains the transient values required to resolve a User Principal.
type UserCredential struct {
	Token          string
	UserID         uint64
	OrganizationID uint64
	ProjectID      uint64
}

// UserPrincipalResolver resolves a verified User aggregate into its immutable Principal.
type UserPrincipalResolver struct {
	authenticationService UserAuthenticationService
}

// NewUserPrincipalResolver constructs a User Principal resolver.
func NewUserPrincipalResolver(authenticationService UserAuthenticationService) *UserPrincipalResolver {
	return &UserPrincipalResolver{authenticationService: authenticationService}
}

// Resolve verifies a User credential and returns the aggregate-owned Principal.
func (resolver *UserPrincipalResolver) Resolve(ctx context.Context, credential UserCredential) (authentication.Principal, error) {
	if !validator.NonNil(resolver) || !validator.NonNil(resolver.authenticationService) {
		return authentication.Principal{}, pkg_errors.AuthenticationResolverInvalidError{}
	}
	if !validator.NonNil(ctx) || !validator.NotBlank(credential.Token) || strings.TrimSpace(credential.Token) != credential.Token ||
		!validator.Between(credential.UserID, uint64(1), uint64(math.MaxInt64)) ||
		!validator.Between(credential.OrganizationID, uint64(0), uint64(math.MaxInt64)) ||
		!validator.Between(credential.ProjectID, uint64(0), uint64(math.MaxInt64)) {
		return authentication.Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	if err := ctx.Err(); err != nil {
		return authentication.Principal{}, err
	}

	value, err := resolver.authenticationService.ResolveUserCredential(ctx, credential.Token, credential.UserID, credential.OrganizationID, credential.ProjectID)
	if err != nil {
		return authentication.Principal{}, err
	}
	if err := ctx.Err(); err != nil {
		return authentication.Principal{}, err
	}
	principal := value.Principal()
	subject := principal.Subject()
	tenant := principal.Tenant()
	organizations := value.Organizations()
	proof := principal.Proof()
	_, delegated := principal.Actor()
	user, hasUser := value.User()
	if credential.OrganizationID == 0 && credential.ProjectID == 0 {
		expectedOrganizationID := uint64(0)
		if len(organizations) == 1 {
			expectedOrganizationID = organizations[0].ID
		}
		if tenant.OrganizationID != expectedOrganizationID {
			return authentication.Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
		}
	}
	if subject.Type != authentication.IdentityUser || subject.ID != credential.UserID || subject.Issuer != "" || delegated ||
		tenant.ProjectID != credential.ProjectID || proof.Method != authentication.AuthenticationUserSession ||
		proof.Audience != authentication.AudienceNone || proof.Reference.Type != authentication.ProofReferenceUserSession ||
		!hasUser || user.ID != credential.UserID || credential.OrganizationID != 0 && tenant.OrganizationID != credential.OrganizationID {
		return authentication.Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	return principal, nil
}
