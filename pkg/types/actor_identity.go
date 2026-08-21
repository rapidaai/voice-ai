package types

import (
	"errors"
	"strconv"
)

type ActorType string

const (
	ActorTypeUser         ActorType = "user"
	ActorTypeProject      ActorType = "project"
	ActorTypeOrganization ActorType = "organization"
	ActorTypeService      ActorType = "service"
	ActorTypeSystem       ActorType = "system"
	ActorTypeUnknown      ActorType = "unknown"
)

type ActorIdentity struct {
	Type ActorType `json:"type"`
	ID   string    `json:"id"`
}

type ActorIdentityProvider interface {
	AuditActor() (ActorIdentity, bool)
}

func ResolveAuditActor(auth AuthenticationPrinciple) (ActorIdentity, error) {
	if err := requireAuthenticated(auth); err != nil {
		return ActorIdentity{}, errors.New("authenticated principle is required for audit actor resolution")
	}
	provider, ok := auth.(ActorIdentityProvider)
	if !ok {
		return ActorIdentity{}, errors.New("authenticated principle does not provide an audit actor")
	}
	actor, ok := provider.AuditActor()
	if !ok || !actor.Type.isDurable() || actor.ID == "" {
		return ActorIdentity{}, errors.New("authenticated principle has no durable audit actor")
	}
	if actor.Type != ActorType(auth.Type()) {
		return ActorIdentity{}, errors.New("audit actor type does not match authenticated principle type")
	}
	return actor, nil
}

func (actorType ActorType) isDurable() bool {
	switch actorType {
	case ActorTypeUser, ActorTypeProject, ActorTypeOrganization, ActorTypeService, ActorTypeSystem:
		return true
	default:
		return false
	}
}

func numericActor(actorType ActorType, id *uint64) (ActorIdentity, bool) {
	if id == nil || *id == 0 {
		return ActorIdentity{}, false
	}
	return ActorIdentity{Type: actorType, ID: strconv.FormatUint(*id, 10)}, true
}
