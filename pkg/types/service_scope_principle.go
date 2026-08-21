// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

/*
Service scope
*/
type ServiceScope struct {
	UserId         *uint64 `json:"userId"`
	ProjectId      *uint64 `json:"projectId"`
	OrganizationId *uint64 `json:"organizationId"`
	CurrentToken   string  `json:"currentToken"`
}

func (ss *ServiceScope) AuditActor() (ActorIdentity, bool) {
	return ActorIdentity{}, false
}

func (ss *ServiceScope) DelegatedContext() (DelegatedContext, bool) {
	if ss.OrganizationId == nil {
		return DelegatedContext{}, false
	}
	return normalizeDelegatedContext(DelegatedContext{
		UserID:         ss.UserId,
		OrganizationID: *ss.OrganizationId,
		ProjectID:      ss.ProjectId,
	}, true)
}

func (ss *ServiceScope) IsAuthenticated() bool {
	_, ok := ss.DelegatedContext()
	return ok
}

func (ss *ServiceScope) Scope(allowed ...AuthType) (AuthenticationPrinciple, error) {
	if !ss.IsAuthenticated() {
		return nil, ErrUnauthenticated
	}
	for _, authType := range allowed {
		if authType == AuthTypeService {
			return ss, nil
		}
	}
	return nil, ErrAuthenticationScopeNotAllowed
}

func (ss *ServiceScope) GetCurrentToken() string {
	return ss.CurrentToken
}

func (ss *ServiceScope) Type() AuthType {
	return AuthTypeService
}
