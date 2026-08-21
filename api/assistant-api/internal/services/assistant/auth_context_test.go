package internal_assistant_service

import (
	"context"
	"testing"

	"github.com/rapidaai/pkg/types"
)

func TestMutationRequiresUserCapability(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}

	service := &assistantToolService{}
	if _, err := service.Create(context.Background(), auth, 33, "tool", nil, nil, "sync", nil); err == nil {
		t.Fatal("Create() error = nil without user capability")
	}
}

func TestScopeReadRequiresProjectCapability(t *testing.T) {
	organizationID := uint64(11)
	auth := &types.Authentication{
		AuthType:          types.AuthTypeOrg,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
	}

	service := &assistantToolService{}
	if _, err := service.GetLog(context.Background(), auth, 22, 33); err == nil {
		t.Fatal("GetLog() error = nil without project capability")
	}
}

func TestScopeReadRejectsMismatchedProject(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}

	service := &assistantToolService{}
	if _, err := service.GetLog(context.Background(), auth, 23, 33); err == nil {
		t.Fatal("GetLog() error = nil for mismatched project")
	}
}
