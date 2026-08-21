package sip_pipeline

import (
	"testing"

	"github.com/rapidaai/pkg/types"
)

func TestReconstructCallContextUsesDelegatedContext(t *testing.T) {
	organizationID := uint64(7)
	projectID := uint64(8)
	auth := &types.Authentication{
		AuthType:          types.AuthTypeService,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}

	callContext, err := reconstructCallContext(auth, 1, 2, "inbound", "call", "context", "sip:from@example.com", "sip:to@example.com")
	if err != nil {
		t.Fatalf("reconstructCallContext() error = %v", err)
	}
	if callContext.OrganizationID != organizationID || callContext.ProjectID != projectID {
		t.Fatalf("reconstructCallContext() = %+v", callContext)
	}
}

func TestReconstructCallContextRejectsInvalidAuthentication(t *testing.T) {
	if _, err := reconstructCallContext(nil, 1, 2, "inbound", "call", "context", "sip:from@example.com", "sip:to@example.com"); err == nil {
		t.Fatal("reconstructCallContext() succeeded without authentication")
	}
}
