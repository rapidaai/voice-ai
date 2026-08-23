// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import "fmt"

/*
Service scope
*/
type ServiceScope struct {
	ActorId           uint64   `json:"actorId"`
	Issuer            string   `json:"issuer"`
	Audience          string   `json:"audience"`
	DelegatedAuthType AuthType `json:"delegatedAuthType"`
	DelegatedActorId  *uint64  `json:"delegatedActorId"`
	ProjectId         *uint64  `json:"projectId"`
	OrganizationId    *uint64  `json:"organizationId"`
	CurrentToken      string   `json:"currentToken"`
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
	_, err := ss.Authentication()
	return err == nil
}

func (ss *ServiceScope) Authentication() (*Authentication, error) {
	if ss == nil || ss.Issuer == "" || ss.Audience != ServiceAssertionAudience {
		return nil, ErrUnauthenticated
	}
	caller, callerOK := ss.AuditActor()
	delegatedContext, contextOK := ss.DelegatedContext()
	if !callerOK || !contextOK {
		return nil, ErrUnauthenticated
	}
	auth := &Authentication{
		AuthType:          AuthTypeService,
		ActorValue:        &caller,
		CallerValue:       &caller,
		OrganizationValue: &OrganizationContext{OrganizationID: delegatedContext.OrganizationID},
	}
	if delegatedContext.ProjectID != nil {
		auth.ProjectValue = &ProjectContext{
			OrganizationID: delegatedContext.OrganizationID,
			ProjectID:      *delegatedContext.ProjectID,
		}
	}
	if ss.DelegatedAuthType == "" && ss.DelegatedActorId == nil {
		return auth, nil
	}
	if ss.DelegatedAuthType == "" || ss.DelegatedActorId == nil {
		return nil, fmt.Errorf("%w: claims are partial", ErrInvalidDelegatedIdentity)
	}
	actor := ActorIdentity{Type: ActorType(ss.DelegatedAuthType), ID: *ss.DelegatedActorId}
	if err := actor.Validate(); err != nil {
		return nil, ErrInvalidDelegatedIdentity
	}
	auth.AuthType = ss.DelegatedAuthType
	auth.ActorValue = &actor
	switch ss.DelegatedAuthType {
	case AuthTypeUser:
		auth.UserValue = &UserContext{UserID: actor.ID}
	case AuthTypeProject:
		if auth.ProjectValue == nil {
			return nil, fmt.Errorf("%w: project context is required", ErrInvalidDelegatedIdentity)
		}
	case AuthTypeOrg:
		if auth.ProjectValue != nil {
			return nil, fmt.Errorf("%w: organization actor forbids project context", ErrInvalidDelegatedIdentity)
		}
	case AuthTypeService, AuthTypeSystem:
	default:
		return nil, ErrUnsupportedDelegatedAuthentication
	}
	if !auth.IsAuthenticated() {
		return nil, ErrUnauthenticated
	}
	return auth, nil
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
