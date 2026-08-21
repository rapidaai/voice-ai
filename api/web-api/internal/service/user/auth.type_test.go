package internal_user_service

import (
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
	if actor != (types.ActorIdentity{Type: types.ActorTypeUser, ID: "73"}) {
		t.Fatalf("AuditActor() = %+v", actor)
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

	userID, err := types.RequireUser(principle)
	if err != nil || userID != 73 {
		t.Fatalf("RequireUser() = %d, %v", userID, err)
	}
	organizationID, err := types.RequireOrganization(principle)
	if err != nil || organizationID != 81 {
		t.Fatalf("RequireOrganization() = %d, %v", organizationID, err)
	}
	projectContext, err := types.RequireProject(principle)
	if err != nil {
		t.Fatalf("RequireProject() error = %v", err)
	}
	if projectContext != (types.ProjectContext{OrganizationID: 81, ProjectID: 92}) {
		t.Fatalf("RequireProject() = %+v", projectContext)
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
	if _, err := types.RequireProject(principle); err == nil {
		t.Fatal("RequireProject() error = nil without selected project")
	}
}
