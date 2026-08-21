package sip_pipeline

import (
	"testing"

	"github.com/rapidaai/pkg/types"
)

func TestReconstructCallContextUsesDelegatedContext(t *testing.T) {
	organizationID := uint64(7)
	projectID := uint64(8)
	auth := &types.ServiceScope{OrganizationId: &organizationID, ProjectId: &projectID, CurrentToken: "token"}

	callContext := reconstructCallContext(auth, 1, 2, "inbound", "call", "context", "sip:from@example.com", "sip:to@example.com")
	if callContext.OrganizationID != organizationID || callContext.ProjectID != projectID {
		t.Fatalf("reconstructCallContext() = %+v", callContext)
	}
}
