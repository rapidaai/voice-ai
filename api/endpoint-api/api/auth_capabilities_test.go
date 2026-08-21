package endpoint_api

import (
	"context"
	"testing"

	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

func TestGetProjectPrincipleGRPC(t *testing.T) {
	organizationID := uint64(10)
	projectID := uint64(20)
	projectScope := &types.ProjectScope{
		OrganizationId: &organizationID,
		ProjectId:      &projectID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}
	projectContext := context.WithValue(
		context.Background(),
		types.CTX_,
		&types.PlainClaimPrinciple[*types.ProjectScope]{Info: projectScope},
	)

	auth, ok := getProjectPrincipleGRPC(projectContext)
	if !ok || auth != projectScope {
		t.Fatalf("getProjectPrincipleGRPC() = %T, %v", auth, ok)
	}
}

func TestGetProjectPrincipleGRPCRejectsMissingProjectCapability(t *testing.T) {
	organizationID := uint64(10)
	organizationScope := &types.OrganizationScope{
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}
	organizationContext := context.WithValue(
		context.Background(),
		types.CTX_,
		&types.PlainClaimPrinciple[*types.OrganizationScope]{Info: organizationScope},
	)

	if auth, ok := getProjectPrincipleGRPC(organizationContext); ok || auth != nil {
		t.Fatalf("getProjectPrincipleGRPC() = %T, %v", auth, ok)
	}
}
