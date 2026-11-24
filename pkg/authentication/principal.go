// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/validator"
)

// PermissionKey identifies one effective route authority.
type PermissionKey string

// IdentityType identifies the effective subject class.
type IdentityType string

const (
	IdentityUser                   IdentityType = "user"
	IdentityProjectCredential      IdentityType = "project"
	IdentityOrganizationCredential IdentityType = "organization"
	IdentityService                IdentityType = "service"
)

// AuthenticationMethod identifies how the immediate caller proved its identity.
type AuthenticationMethod string

const (
	AuthenticationUserSession            AuthenticationMethod = "user_session"
	AuthenticationProjectCredential      AuthenticationMethod = "project_credential"
	AuthenticationOrganizationCredential AuthenticationMethod = "organization_credential"
	AuthenticationServiceAssertion       AuthenticationMethod = "service_assertion"
)

// Audience constrains the verified proof to its intended use.
type Audience string

const (
	AudienceNone           Audience = ""
	AudienceManagement     Audience = "management"
	AudienceRapidaInternal Audience = "rapida_internal"
)

// ProofReferenceType identifies the nonsecret proof correlation value.
type ProofReferenceType string

const (
	ProofReferenceUserSession ProofReferenceType = "user_session"
	ProofReferenceCredential  ProofReferenceType = "credential"
	ProofReferenceSigningKey  ProofReferenceType = "signing_key"
)

// Identity identifies the effective subject.
type Identity struct {
	Type   IdentityType
	ID     uint64
	Issuer string
}

// ServiceActor identifies the verified immediate service caller.
type ServiceActor struct {
	ID     uint64
	Issuer string
}

// Tenant identifies the single Organization or Project selected for the request.
type Tenant struct {
	OrganizationID uint64
	ProjectID      uint64
}

// Proof records nonsecret facts about verified caller authentication.
type Proof struct {
	Method    AuthenticationMethod
	Audience  Audience
	Reference ProofReference
}

// ProofReference carries a stable nonsecret audit correlation value.
type ProofReference struct {
	Type  ProofReferenceType
	Value string
}

// Principal is an immutable authenticated request identity and authority snapshot.
type Principal struct {
	subject     Identity
	actor       *ServiceActor
	tenant      Tenant
	proof       Proof
	authorities []PermissionKey
}

// NewDirectPrincipal constructs a principal authenticated by its subject's credential.
func NewDirectPrincipal(subject Identity, tenant Tenant, proof Proof, authorities []PermissionKey) (Principal, error) {
	if !validator.Between(subject.ID, uint64(1), uint64(math.MaxInt64)) ||
		!validator.Between(tenant.OrganizationID, uint64(0), uint64(math.MaxInt64)) ||
		!validator.Between(tenant.ProjectID, uint64(0), uint64(math.MaxInt64)) ||
		(!validator.NonZero(tenant.OrganizationID) && validator.NonZero(tenant.ProjectID)) {
		return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	referenceLength := utf8.RuneCountInString(proof.Reference.Value)
	if !utf8.ValidString(proof.Reference.Value) || !validator.NotBlank(proof.Reference.Value) ||
		strings.TrimSpace(proof.Reference.Value) != proof.Reference.Value || strings.ContainsRune(proof.Reference.Value, '\x00') ||
		!validator.Between(referenceLength, 1, 200) {
		return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}

	switch subject.Type {
	case IdentityUser:
		if subject.Issuer != "" || proof.Method != AuthenticationUserSession ||
			proof.Audience != AudienceNone || proof.Reference.Type != ProofReferenceUserSession {
			return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
		}
	case IdentityProjectCredential:
		if subject.Issuer != "" || !validator.AllNonZero(tenant.OrganizationID, tenant.ProjectID) ||
			proof.Method != AuthenticationProjectCredential || proof.Audience != AudienceManagement ||
			proof.Reference.Type != ProofReferenceCredential || proof.Reference.Value != strconv.FormatUint(subject.ID, 10) {
			return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
		}
	case IdentityOrganizationCredential:
		if subject.Issuer != "" || !validator.NonZero(tenant.OrganizationID) || validator.NonZero(tenant.ProjectID) ||
			proof.Method != AuthenticationOrganizationCredential || proof.Audience != AudienceNone ||
			proof.Reference.Type != ProofReferenceCredential || proof.Reference.Value != strconv.FormatUint(subject.ID, 10) {
			return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
		}
	case IdentityService:
		if !utf8.ValidString(subject.Issuer) || !validator.NotBlank(subject.Issuer) ||
			strings.TrimSpace(subject.Issuer) != subject.Issuer || strings.ContainsRune(subject.Issuer, '\x00') ||
			proof.Method != AuthenticationServiceAssertion || proof.Audience != AudienceRapidaInternal ||
			proof.Reference.Type != ProofReferenceSigningKey {
			return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
		}
	default:
		return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}

	principal := Principal{subject: subject, tenant: tenant, proof: proof}
	principal, ok := principal.withAuthorities(authorities)
	if !ok {
		return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	return principal, nil
}

// NewDelegatedPrincipal constructs a subject asserted by a verified Service actor.
func NewDelegatedPrincipal(subject Identity, actor ServiceActor, tenant Tenant, proof Proof, authorities []PermissionKey) (Principal, error) {
	if !validator.Between(subject.ID, uint64(1), uint64(math.MaxInt64)) || subject.Issuer != "" ||
		!validator.Between(actor.ID, uint64(1), uint64(math.MaxInt64)) || !utf8.ValidString(actor.Issuer) ||
		!validator.NotBlank(actor.Issuer) || strings.TrimSpace(actor.Issuer) != actor.Issuer || strings.ContainsRune(actor.Issuer, '\x00') {
		return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	referenceLength := utf8.RuneCountInString(proof.Reference.Value)
	if !validator.Between(tenant.OrganizationID, uint64(0), uint64(math.MaxInt64)) ||
		!validator.Between(tenant.ProjectID, uint64(0), uint64(math.MaxInt64)) ||
		(!validator.NonZero(tenant.OrganizationID) && validator.NonZero(tenant.ProjectID)) ||
		proof.Method != AuthenticationServiceAssertion || proof.Audience != AudienceRapidaInternal ||
		proof.Reference.Type != ProofReferenceSigningKey || !utf8.ValidString(proof.Reference.Value) ||
		!validator.NotBlank(proof.Reference.Value) || strings.TrimSpace(proof.Reference.Value) != proof.Reference.Value ||
		strings.ContainsRune(proof.Reference.Value, '\x00') || !validator.Between(referenceLength, 1, 200) {
		return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}

	switch subject.Type {
	case IdentityUser:
	case IdentityProjectCredential:
		if !validator.AllNonZero(tenant.OrganizationID, tenant.ProjectID) {
			return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
		}
	case IdentityOrganizationCredential:
		if !validator.NonZero(tenant.OrganizationID) || validator.NonZero(tenant.ProjectID) {
			return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
		}
	default:
		return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}

	actorCopy := actor
	principal := Principal{subject: subject, actor: &actorCopy, tenant: tenant, proof: proof}
	principal, ok := principal.withAuthorities(authorities)
	if !ok {
		return Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	return principal, nil
}

// Subject returns the effective request identity.
func (principal Principal) Subject() Identity {
	return principal.subject
}

// Actor returns the verified immediate service caller when the subject is delegated.
func (principal Principal) Actor() (ServiceActor, bool) {
	if principal.actor == nil {
		return ServiceActor{}, false
	}
	return *principal.actor, true
}

// Caller returns the immediate caller for policy and audit decisions.
func (principal Principal) Caller() Identity {
	if principal.actor == nil {
		return principal.subject
	}
	return Identity{Type: IdentityService, ID: principal.actor.ID, Issuer: principal.actor.Issuer}
}

// Tenant returns the request's selected tenant.
func (principal Principal) Tenant() Tenant {
	return principal.tenant
}

// Proof returns the verified authentication proof facts.
func (principal Principal) Proof() Proof {
	return principal.proof
}

// Authorities returns a copy of the effective permission snapshot.
func (principal Principal) Authorities() []PermissionKey {
	return append([]PermissionKey(nil), principal.authorities...)
}

func (principal Principal) withAuthorities(authorities []PermissionKey) (Principal, bool) {
	unique := make(map[PermissionKey]struct{}, len(authorities))
	for _, authority := range authorities {
		value := string(authority)
		if !utf8.ValidString(value) || !validator.NotBlank(value) || strings.TrimSpace(value) != value || strings.ContainsRune(value, '\x00') {
			return Principal{}, false
		}
		switch principal.subject.Type {
		case IdentityUser:
			if !validator.NonZero(principal.tenant.OrganizationID) ||
				(!validator.NonZero(principal.tenant.ProjectID) && !strings.HasPrefix(value, "organization.")) ||
				(validator.NonZero(principal.tenant.ProjectID) && !strings.HasPrefix(value, "organization.") && !strings.HasPrefix(value, "project.")) {
				return Principal{}, false
			}
		case IdentityProjectCredential:
			if !strings.HasPrefix(value, "project.") {
				return Principal{}, false
			}
		case IdentityOrganizationCredential, IdentityService:
			return Principal{}, false
		default:
			return Principal{}, false
		}
		unique[authority] = struct{}{}
	}
	result := make([]PermissionKey, 0, len(unique))
	for authority := range unique {
		result = append(result, authority)
	}
	sort.Slice(result, func(left int, right int) bool {
		return result[left] < result[right]
	})
	principal.authorities = result
	return principal, true
}
