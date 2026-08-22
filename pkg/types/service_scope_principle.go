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
	ActorId        uint64  `json:"actorId"`
	Issuer         string  `json:"issuer"`
	Audience       string  `json:"audience"`
	ProjectId      *uint64 `json:"projectId"`
	OrganizationId *uint64 `json:"organizationId"`
	CurrentToken   string  `json:"currentToken"`
}

func (ss *ServiceScope) AuditActor() (ActorIdentity, bool) {
	if ss == nil {
		return ActorIdentity{}, false
	}
	actor := ActorIdentity{Type: ActorTypeService, ID: ss.ActorId}
	return actor, actor.Validate() == nil
}

func (ss *ServiceScope) DelegatedContext() (DelegatedContext, bool) {
	if ss.OrganizationId == nil {
		return DelegatedContext{}, false
	}
	return normalizeDelegatedContext(DelegatedContext{
		OrganizationID: *ss.OrganizationId,
		ProjectID:      ss.ProjectId,
	}, true)
}

func (ss *ServiceScope) IsAuthenticated() bool {
	_, contextOK := ss.DelegatedContext()
	_, actorOK := ss.AuditActor()
	return contextOK && actorOK && ss.Issuer != "" && ss.Audience != ""
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
