package internal_assistant_service

import (
	"context"
	"testing"

	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

func TestMutationRequiresUserCapability(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	auth := &types.ProjectScope{
		OrganizationId: &organizationID,
		ProjectId:      &projectID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}

	service := &assistantToolService{}
	if _, err := service.Create(context.Background(), auth, 33, "tool", nil, nil, "sync", nil); err == nil {
		t.Fatal("Create() error = nil without user capability")
	}
}

func TestScopeReadRequiresProjectCapability(t *testing.T) {
	organizationID := uint64(11)
	auth := &types.OrganizationScope{
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}

	service := &assistantToolService{}
	if _, err := service.GetLog(context.Background(), auth, 22, 33); err == nil {
		t.Fatal("GetLog() error = nil without project capability")
	}
}

func TestScopeReadRejectsMismatchedProject(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	auth := &types.ProjectScope{
		OrganizationId: &organizationID,
		ProjectId:      &projectID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}

	service := &assistantToolService{}
	if _, err := service.GetLog(context.Background(), auth, 23, 33); err == nil {
		t.Fatal("GetLog() error = nil for mismatched project")
	}
}
