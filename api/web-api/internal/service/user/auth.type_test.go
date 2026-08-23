package internal_user_service

import (
	"math"
	"testing"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/types"
)

var _ types.Principle = (*authPrinciple)(nil)
var _ types.ProjectContextProvider = (*authPrinciple)(nil)

func TestAuthPrincipleAuditActor(t *testing.T) {
	principle := &authPrinciple{user: &internal_entity.UserAuth{}}
	principle.user.Id = 73

	actor, ok := principle.AuditActor()
	if !ok {
		t.Fatal("AuditActor() ok = false, want true")
	}
	if actor != (types.ActorIdentity{Type: types.ActorTypeUser, ID: 73}) {
		t.Fatalf("AuditActor() = %+v", actor)
	}
}

func TestAuthPrincipleAuditActorRange(t *testing.T) {
	tests := []struct {
		name string
		id   uint64
		ok   bool
	}{
		{name: "zero rejected", id: 0},
		{name: "max bigint accepted", id: math.MaxInt64, ok: true},
		{name: "above max bigint rejected", id: uint64(math.MaxInt64) + 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			principle := &authPrinciple{user: &internal_entity.UserAuth{}}
			principle.user.Id = test.id
			actor, ok := principle.AuditActor()
			if ok != test.ok {
				t.Fatalf("AuditActor() ok = %v, want %v", ok, test.ok)
			}
			if ok && actor.ID != test.id {
				t.Fatalf("AuditActor() ID = %d, want %d", actor.ID, test.id)
			}
		})
	}
}

func TestAuthPrincipleCapabilities(t *testing.T) {
	user := &internal_entity.UserAuth{}
	user.Id = 73
	principle := &authPrinciple{
		user: user,
		userOrgRole: &internal_entity.UserOrganizationRole{
			OrganizationId: 81,
		},
		currentProjectRole: &types.ProjectRole{ProjectId: 92},
	}

	userID, ok := principle.UserIdentity()
	if !ok || userID != 73 {
		t.Fatalf("UserIdentity() = %d, %v", userID, ok)
	}
	organizationID, ok := principle.OrganizationContext()
	if !ok || organizationID != 81 {
		t.Fatalf("OrganizationContext() = %d, %v", organizationID, ok)
	}
	projectContext, ok := principle.ProjectContext()
	if !ok {
		t.Fatal("ProjectContext() ok = false")
	}
	if projectContext != (types.ProjectContext{OrganizationID: 81, ProjectID: 92}) {
		t.Fatalf("ProjectContext() = %+v", projectContext)
	}
}

func TestAuthPrincipleProjectContextOptional(t *testing.T) {
	user := &internal_entity.UserAuth{}
	user.Id = 73
	principle := &authPrinciple{
		user: user,
		userOrgRole: &internal_entity.UserOrganizationRole{
			OrganizationId: 81,
		},
	}

	if !principle.IsAuthenticated() {
		t.Fatal("IsAuthenticated() = false without selected project")
	}
	if _, ok := principle.ProjectContext(); ok {
		t.Fatal("ProjectContext() ok = true without selected project")
	}
}
