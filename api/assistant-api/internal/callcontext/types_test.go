package internal_callcontext

import (
	"fmt"
	"math"
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
				ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: 11},
				UserValue:         &types.UserContext{UserID: 11},
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
				ProjectValue:      &types.ProjectContext{OrganizationID: 33, ProjectID: 22},
			},
		},
		{
			name: "project",
			auth: &types.Authentication{
				AuthType:          types.AuthTypeProject,
				ActorValue:        &types.ActorIdentity{Type: types.ActorTypeProject, ID: 44},
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
				ProjectValue:      &types.ProjectContext{OrganizationID: 33, ProjectID: 22},
			},
		},
		{
			name: "organization",
			auth: &types.Authentication{
				AuthType:          types.AuthTypeOrg,
				ActorValue:        &types.ActorIdentity{Type: types.ActorTypeOrganization, ID: 55},
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
			},
		},
		{
			name: "service",
			auth: &types.Authentication{
				AuthType:          types.AuthTypeService,
				ActorValue:        &types.ActorIdentity{Type: types.ActorTypeService, ID: 66},
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

func TestCallContextRejectsActorlessAuthenticatedSnapshots(t *testing.T) {
	tests := []struct {
		name string
		auth *types.Authentication
	}{
		{
			name: "user",
			auth: &types.Authentication{
				AuthType:          types.AuthTypeUser,
				UserValue:         &types.UserContext{UserID: 11},
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
			},
		},
		{
			name: "project",
			auth: &types.Authentication{
				AuthType:          types.AuthTypeProject,
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
				ProjectValue:      &types.ProjectContext{OrganizationID: 33, ProjectID: 22},
			},
		},
		{
			name: "organization",
			auth: &types.Authentication{
				AuthType:          types.AuthTypeOrg,
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
			},
		},
		{
			name: "service",
			auth: &types.Authentication{
				AuthType:          types.AuthTypeService,
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := (&CallContext{}).SetAuthentication(test.auth); err == nil {
				t.Fatal("SetAuthentication() error = nil, want actor error")
			}
		})
	}
}

func TestCallContextStoredActorlessSnapshotsFailClosed(t *testing.T) {
	userID := uint64(11)
	tests := []struct {
		name        string
		callContext *CallContext
	}{
		{
			name: "user",
			callContext: &CallContext{
				AuthType:       types.AuthTypeUser.String(),
				AuthUserID:     &userID,
				OrganizationID: 33,
			},
		},
		{
			name: "project",
			callContext: &CallContext{
				AuthType:       types.AuthTypeProject.String(),
				OrganizationID: 33,
				ProjectID:      22,
			},
		},
		{
			name: "organization",
			callContext: &CallContext{
				AuthType:       types.AuthTypeOrg.String(),
				OrganizationID: 33,
			},
		},
		{
			name: "service",
			callContext: &CallContext{
				AuthType:       types.AuthTypeService.String(),
				OrganizationID: 33,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.callContext.ToAuthentication(); err == nil {
				t.Fatal("ToAuthentication() error = nil, want actor error")
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
	if err == nil {
		t.Fatal("ToAuthentication() error = nil, want incomplete actor error")
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

func TestCallContextActorRange(t *testing.T) {
	tests := []struct {
		name      string
		actorID   uint64
		wantError bool
	}{
		{name: "zero rejected", actorID: 0, wantError: true},
		{name: "max bigint accepted", actorID: math.MaxInt64},
		{name: "above max bigint rejected", actorID: uint64(math.MaxInt64) + 1, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth := &types.Authentication{
				AuthType:          types.AuthTypeUser,
				ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: test.actorID},
				UserValue:         &types.UserContext{UserID: test.actorID},
				OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
			}
			callContext := &CallContext{}

			err := callContext.SetAuthentication(auth)
			if test.wantError {
				if err == nil {
					t.Fatal("SetAuthentication() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("SetAuthentication() error = %v", err)
			}
			reconstructed, err := callContext.ToAuthentication()
			if err != nil {
				t.Fatalf("ToAuthentication() error = %v", err)
			}
			actor := reconstructed.Actor()
			if actor.ID != test.actorID {
				t.Fatalf("Actor() = %+v", actor)
			}
		})
	}
}

func TestCallContextStoredActorRangeFailsClosed(t *testing.T) {
	for _, actorID := range []uint64{0, uint64(math.MaxInt64) + 1} {
		t.Run(fmt.Sprintf("actor id %d", actorID), func(t *testing.T) {
			actorType := string(types.ActorTypeUser)
			userID := uint64(11)
			callContext := &CallContext{
				AuthType:       types.AuthTypeUser.String(),
				AuthUserID:     &userID,
				OrganizationID: 33,
				AuthActorType:  &actorType,
				AuthActorID:    &actorID,
			}

			if _, err := callContext.ToAuthentication(); err == nil {
				t.Fatal("ToAuthentication() error = nil, want error")
			}
		})
	}
}
