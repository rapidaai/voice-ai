package internal_callcontext

import (
	"errors"
	"testing"

	"github.com/rapidaai/pkg/types"
)

func TestCallContextAuthenticationSnapshotRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		auth *types.Authentication
	}{
		{
			name: "user",
			auth: &types.Authentication{
				AuthType:          types.AuthTypeUser,
				ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: "11"},
				UserValue:         &types.UserContext{UserID: 11},
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
				ProjectValue:      &types.ProjectContext{OrganizationID: 33, ProjectID: 22},
			},
		},
		{
			name: "project",
			auth: &types.Authentication{
				AuthType:          types.AuthTypeProject,
				ActorValue:        &types.ActorIdentity{Type: types.ActorTypeProject, ID: "44"},
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
				ProjectValue:      &types.ProjectContext{OrganizationID: 33, ProjectID: 22},
			},
		},
		{
			name: "service",
			auth: &types.Authentication{
				AuthType:          types.AuthTypeService,
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
				ProjectValue:      &types.ProjectContext{OrganizationID: 33, ProjectID: 22},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callContext := &CallContext{}
			if err := callContext.SetAuthentication(test.auth); err != nil {
				t.Fatalf("SetAuthentication() error = %v", err)
			}
			reconstructed, err := callContext.ToAuthentication()
			if err != nil {
				t.Fatalf("ToAuthentication() error = %v", err)
			}
			if reconstructed.AuthType != test.auth.AuthType {
				t.Fatalf("AuthType = %q, want %q", reconstructed.AuthType, test.auth.AuthType)
			}
			if test.auth.UserValue != nil {
				userContext, err := reconstructed.UserContext()
				if err != nil || userContext.UserID != test.auth.UserValue.UserID {
					t.Fatalf("UserContext() = %+v, %v", userContext, err)
				}
			}
			organizationContext, err := reconstructed.OrganizationContext()
			if err != nil || organizationContext.OrganizationID != test.auth.OrganizationValue.OrganizationID {
				t.Fatalf("OrganizationContext() = %+v, %v", organizationContext, err)
			}
			if test.auth.ProjectValue != nil {
				projectContext, err := reconstructed.ProjectContext()
				if err != nil || projectContext != *test.auth.ProjectValue {
					t.Fatalf("ProjectContext() = %+v, %v", projectContext, err)
				}
			}
		})
	}
}

func TestCallContextOldUserSnapshotWithoutUserIDFailsClosed(t *testing.T) {
	callContext := &CallContext{
		AuthType:       types.AuthTypeUser.String(),
		OrganizationID: 33,
		ProjectID:      22,
	}

	_, err := callContext.ToAuthentication()
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("ToAuthentication() error = %v, want ErrUnauthenticated", err)
	}
}

func TestCallContextIncompleteActorFailsClosed(t *testing.T) {
	actorType := string(types.ActorTypeService)
	callContext := &CallContext{
		AuthType:       types.AuthTypeService.String(),
		OrganizationID: 33,
		AuthActorType:  &actorType,
	}

	if _, err := callContext.ToAuthentication(); err == nil {
		t.Fatal("ToAuthentication() error = nil, want incomplete actor error")
	}
}
