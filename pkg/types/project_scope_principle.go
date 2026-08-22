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

type ProjectScope struct {
	CredentialId   *uint64 `json:"credentialId" gorm:"column:credential_id"`
	ProjectId      *uint64 `json:"projectId"`
	OrganizationId *uint64 `json:"organizationId"`
	Status         string  `json:"status"`
	CurrentToken   string  `json:"currentToken"`
}

func (ss *ProjectScope) AuditActor() (ActorIdentity, bool) {
	if ss.CredentialId == nil || *ss.CredentialId == 0 || *ss.CredentialId > math.MaxInt64 {
		return ActorIdentity{}, false
	}
	return ActorIdentity{Type: ActorTypeProject, ID: *ss.CredentialId}, true
}

func (ss *ProjectScope) OrganizationContext() (uint64, bool) {
	if ss.OrganizationId == nil || *ss.OrganizationId == 0 {
		return 0, false
	}
	return *ss.OrganizationId, true
}

func (ss *ProjectScope) ProjectContext() (ProjectContext, bool) {
	organizationID, ok := ss.OrganizationContext()
	if !ok || ss.ProjectId == nil || *ss.ProjectId == 0 {
		return ProjectContext{}, false
	}
	return ProjectContext{OrganizationID: organizationID, ProjectID: *ss.ProjectId}, true
}

func (ss *ProjectScope) IsActive() bool {
	return ss.Status == type_enums.RECORD_ACTIVE.String()
}

func (ss *ProjectScope) IsAuthenticated() bool {
	_, ok := ss.ProjectContext()
	return ok && ss.IsActive()
}

func (ss *ProjectScope) Scope(allowed ...AuthType) (AuthenticationPrinciple, error) {
	if !ss.IsAuthenticated() {
		return nil, ErrUnauthenticated
	}
	for _, authType := range allowed {
		if authType == AuthTypeProject {
			return ss, nil
		}
	}
	return nil, ErrAuthenticationScopeNotAllowed
}

func (ss *ProjectScope) GetCurrentToken() string {
	return ss.CurrentToken
}

func (aP *ProjectScope) Type() AuthType {
	return AuthTypeProject
}
