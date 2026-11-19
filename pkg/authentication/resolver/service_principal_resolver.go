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
	authentication_jwt "github.com/rapidaai/pkg/authentication/jwt"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/validator"
)

// ServiceAssertionCredential contains the transient service assertion presented by a request.
type ServiceAssertionCredential struct {
	Assertion string
}

// ServicePrincipalResolver is the sole owner of Service principal construction.
type ServicePrincipalResolver struct {
	verifier authentication_jwt.Verifier
}

// NewServicePrincipalResolver constructs the authoritative Service verifier and resolver.
func NewServicePrincipalResolver(
	audience string,
	key []byte,
) *ServicePrincipalResolver {
	return &ServicePrincipalResolver{verifier: authentication_jwt.NewVerifier(audience, key)}
}

// Resolve verifies a Service assertion and creates its immutable request Principal.
func (resolver *ServicePrincipalResolver) Resolve(ctx context.Context, credential ServiceAssertionCredential) (authentication.Principal, error) {
	if !validator.NonNil(resolver) {
		return authentication.Principal{}, pkg_errors.AuthenticationResolverInvalidError{}
	}
	if !validator.NonNil(ctx) || !validator.NotBlank(credential.Assertion) || strings.TrimSpace(credential.Assertion) != credential.Assertion {
		return authentication.Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	if err := ctx.Err(); err != nil {
		return authentication.Principal{}, err
	}
	verified, err := resolver.verifier.Verify(credential.Assertion)
	if err != nil {
		return authentication.Principal{}, err
	}
	if err := ctx.Err(); err != nil {
		return authentication.Principal{}, err
	}
	proof := authentication.Proof{
		Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal,
		Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: verified.KeyFingerprint()},
	}
	if verified.Subject().Type == authentication.IdentityService {
		return authentication.NewDirectPrincipal(verified.Subject(), verified.Tenant(), proof, verified.Authorities())
	}
	return authentication.NewDelegatedPrincipal(verified.Subject(), verified.Actor(), verified.Tenant(), proof, verified.Authorities())
}
