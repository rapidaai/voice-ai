// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authorizer

import (
	"strconv"

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

// RouteAuthorizer evaluates one immutable Principal against one typed route policy.
type RouteAuthorizer struct{}

// Authorize evaluates public, disabled, authenticated, permission, and alternative admission.
func (RouteAuthorizer) Authorize(principal *authentication.Principal, policy RoutePolicy) error {
	if policy.Validate() != nil {
		return pkg_errors.RouteAuthorizationDeniedError{}
	}
	if policy.Authorization == AuthorizationPublic {
		return nil
	}
	if policy.Authorization == AuthorizationDisabled {
		return pkg_errors.RouteAuthorizationDisabledError{}
	}
	if principal == nil || !policy.authorizes(*principal) {
		return pkg_errors.RouteAuthorizationDeniedError{}
	}
	return nil
}

func (policy RoutePolicy) authorizes(principal authentication.Principal) bool {
	if policy.Authorization == AuthorizationAnyOf {
		for _, alternative := range policy.AnyOf {
			authorization := AuthorizationPermission
			if alternative.Authenticated {
				authorization = AuthorizationAuthenticated
			}
			if (RoutePolicy{
				Authorization:  authorization,
				Actors:         alternative.Actors,
				CallerServices: alternative.CallerServices,
				Context:        alternative.Context,
				ProjectToken:   alternative.ProjectToken,
				Permissions:    alternative.Permissions,
			}).authorizes(principal) {
				return true
			}
		}
		return false
	}

	subject := principal.Subject()
	if subject.ID == 0 {
		return false
	}
	actorAllowed := false
	for _, actor := range policy.Actors {
		if subject.Type == actor {
			actorAllowed = true
			break
		}
	}
	if !actorAllowed {
		return false
	}

	if len(policy.CallerServices) > 0 {
		caller := principal.Caller()
		if caller.Type != authentication.IdentityService {
			return false
		}
		callerAllowed := false
		for _, allowed := range policy.CallerServices {
			if caller.ID == allowed.ActorID && caller.Issuer == allowed.Issuer {
				callerAllowed = true
				break
			}
		}
		if !callerAllowed {
			return false
		}
	}

	if policy.ProjectToken != nil && subject.Type == authentication.IdentityProjectCredential {
		tenant := principal.Tenant()
		if tenant.OrganizationID == 0 || tenant.ProjectID == 0 {
			return false
		}
		proof := principal.Proof()
		_, delegated := principal.Actor()
		management := !delegated && proof.Method == authentication.AuthenticationProjectCredential &&
			proof.Audience == authentication.AudienceManagement && proof.Reference.Type == authentication.ProofReferenceCredential &&
			proof.Reference.Value == strconv.FormatUint(subject.ID, 10)
		if policy.ProjectToken.Audience == ProjectTokenAudienceManagement && !management {
			return false
		}
		if policy.ProjectToken.Audience == ProjectTokenAudienceRuntime {
			runtime := delegated && proof.Method == authentication.AuthenticationServiceAssertion &&
				proof.Audience == authentication.AudienceRapidaInternal && proof.Reference.Type == authentication.ProofReferenceSigningKey
			if !management && !runtime {
				return false
			}
		}
	}

	tenant := principal.Tenant()
	switch policy.Context {
	case ContextUser:
		if subject.Type != authentication.IdentityUser {
			return false
		}
	case ContextOrganization:
		if tenant.OrganizationID == 0 {
			return false
		}
		if subject.Type != authentication.IdentityUser && subject.Type != authentication.IdentityOrganizationCredential &&
			subject.Type != authentication.IdentityService {
			return false
		}
	case ContextProject:
		if tenant.OrganizationID == 0 || tenant.ProjectID == 0 {
			return false
		}
		if subject.Type != authentication.IdentityUser && subject.Type != authentication.IdentityProjectCredential &&
			subject.Type != authentication.IdentityService {
			return false
		}
	case ContextService:
		if subject.Type != authentication.IdentityService || tenant.OrganizationID != 0 || tenant.ProjectID != 0 {
			return false
		}
		if _, delegated := principal.Actor(); delegated {
			return false
		}
		proof := principal.Proof()
		if proof.Method != authentication.AuthenticationServiceAssertion ||
			proof.Audience != authentication.AudienceRapidaInternal ||
			proof.Reference.Type != authentication.ProofReferenceSigningKey {
			return false
		}
	default:
		return false
	}

	if policy.Authorization == AuthorizationPermission {
		authorities := principal.Authorities()
		available := make(map[authentication.PermissionKey]struct{}, len(authorities))
		for _, authority := range authorities {
			available[authority] = struct{}{}
		}
		for _, permission := range policy.Permissions {
			if _, exists := available[permission]; !exists {
				return false
			}
		}
	}
	return true
}
