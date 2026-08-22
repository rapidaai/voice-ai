// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"math"

	type_enums "github.com/rapidaai/pkg/types/enums"
)

type OrganizationScope struct {
	CredentialId   *uint64 `json:"credentialId"`
	OrganizationId *uint64 `json:"organizationId"`
	Status         string  `json:"status"`
	CurrentToken   string  `json:"currentToken"`
}

func (ss *OrganizationScope) AuditActor() (ActorIdentity, bool) {
	if ss.CredentialId == nil || *ss.CredentialId == 0 || *ss.CredentialId > math.MaxInt64 {
		return ActorIdentity{}, false
	}
	return ActorIdentity{Type: ActorTypeOrganization, ID: *ss.CredentialId}, true
}

func (ss *OrganizationScope) OrganizationContext() (uint64, bool) {
	if ss.OrganizationId == nil || *ss.OrganizationId == 0 {
		return 0, false
	}
	return *ss.OrganizationId, true
}

func (ss *OrganizationScope) IsActive() bool {
	return ss.Status == type_enums.RECORD_ACTIVE.String()
}

func (ss *OrganizationScope) IsAuthenticated() bool {
	_, organizationOK := ss.OrganizationContext()
	_, actorOK := ss.AuditActor()
	return organizationOK && actorOK && ss.IsActive()
}

func (ss *OrganizationScope) Scope(allowed ...AuthType) (AuthenticationPrinciple, error) {
	if !ss.IsAuthenticated() {
		return nil, ErrUnauthenticated
	}
	for _, authType := range allowed {
		if authType == AuthTypeOrg {
			return ss, nil
		}
	}
	return nil, ErrAuthenticationScopeNotAllowed
}

func (ss *OrganizationScope) GetCurrentToken() string {
	return ss.CurrentToken
}

func (aP *OrganizationScope) Type() AuthType {
	return AuthTypeOrg
}
