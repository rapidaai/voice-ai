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

// OrganizationAuthenticationService verifies one Organization credential and returns its canonical aggregate.
type OrganizationAuthenticationService interface {
	ResolveOrganizationCredential(ctx context.Context, token string) (authentication.Authentication, error)
}

// OrganizationCredential contains the transient Organization credential presented by a request.
type OrganizationCredential struct {
	Token string
}

// OrganizationPrincipalResolver resolves a verified Organization aggregate into its immutable Principal.
type OrganizationPrincipalResolver struct {
	authenticationService OrganizationAuthenticationService
}

// NewOrganizationPrincipalResolver constructs an Organization Principal resolver.
func NewOrganizationPrincipalResolver(authenticationService OrganizationAuthenticationService) *OrganizationPrincipalResolver {
	return &OrganizationPrincipalResolver{authenticationService: authenticationService}
}

// Resolve verifies an Organization credential and returns the aggregate-owned Principal.
func (resolver *OrganizationPrincipalResolver) Resolve(ctx context.Context, credential OrganizationCredential) (authentication.Principal, error) {
	if !validator.NonNil(resolver) || !validator.NonNil(resolver.authenticationService) {
		return authentication.Principal{}, pkg_errors.AuthenticationResolverInvalidError{}
	}
	if !validator.NonNil(ctx) || !validator.NotBlank(credential.Token) || strings.TrimSpace(credential.Token) != credential.Token {
		return authentication.Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	if err := ctx.Err(); err != nil {
		return authentication.Principal{}, err
	}

	value, err := resolver.authenticationService.ResolveOrganizationCredential(ctx, credential.Token)
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
	if subject.Type != authentication.IdentityOrganizationCredential || subject.Issuer != "" || delegated ||
		!validator.AllNonZero(subject.ID, tenant.OrganizationID) || validator.NonZero(tenant.ProjectID) ||
		proof.Method != authentication.AuthenticationOrganizationCredential || proof.Audience != authentication.AudienceNone ||
		proof.Reference.Type != authentication.ProofReferenceCredential {
		return authentication.Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	return principal, nil
}
