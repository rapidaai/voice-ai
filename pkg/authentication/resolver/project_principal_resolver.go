// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package resolver

import (
	"context"
	"strings"

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/validator"
)

// ProjectAuthenticationService verifies one Project credential and returns its canonical aggregate.
type ProjectAuthenticationService interface {
	ResolveProjectCredential(ctx context.Context, token string) (authentication.Authentication, error)
}

// ProjectCredential contains the transient Project credential presented by a request.
type ProjectCredential struct {
	Token string
}

// ProjectPrincipalResolver resolves a verified Project aggregate into its immutable Principal.
type ProjectPrincipalResolver struct {
	authenticationService ProjectAuthenticationService
}

// NewProjectPrincipalResolver constructs a Project Principal resolver.
func NewProjectPrincipalResolver(authenticationService ProjectAuthenticationService) *ProjectPrincipalResolver {
	return &ProjectPrincipalResolver{authenticationService: authenticationService}
}

// Resolve verifies a Project credential and returns the aggregate-owned Principal.
func (resolver *ProjectPrincipalResolver) Resolve(ctx context.Context, credential ProjectCredential) (authentication.Principal, error) {
	if !validator.NonNil(resolver) || !validator.NonNil(resolver.authenticationService) {
		return authentication.Principal{}, pkg_errors.AuthenticationResolverInvalidError{}
	}
	if !validator.NonNil(ctx) || !validator.NotBlank(credential.Token) || strings.TrimSpace(credential.Token) != credential.Token {
		return authentication.Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	if err := ctx.Err(); err != nil {
		return authentication.Principal{}, err
	}

	value, err := resolver.authenticationService.ResolveProjectCredential(ctx, credential.Token)
	if err != nil {
		return authentication.Principal{}, err
	}
	if err := ctx.Err(); err != nil {
		return authentication.Principal{}, err
	}
	principal := value.Principal()
	subject := principal.Subject()
	tenant := principal.Tenant()
	proof := principal.Proof()
	_, delegated := principal.Actor()
	if subject.Type != authentication.IdentityProjectCredential || subject.Issuer != "" || delegated ||
		!validator.AllNonZero(subject.ID, tenant.OrganizationID, tenant.ProjectID) ||
		proof.Method != authentication.AuthenticationProjectCredential || proof.Audience != authentication.AudienceManagement ||
		proof.Reference.Type != authentication.ProofReferenceCredential {
		return authentication.Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	return principal, nil
}
