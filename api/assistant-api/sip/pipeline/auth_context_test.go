package sip_pipeline

import (
	"testing"

	"github.com/rapidaai/pkg/types"
)

func TestReconstructCallContextUsesDelegatedContext(t *testing.T) {
	organizationID := uint64(7)
	projectID := uint64(8)
	serviceID := uint64(9)
	auth := &types.Authentication{
		AuthType:          types.AuthTypeService,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeService, ID: serviceID},
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
	if callContext.AuthActorType == nil || *callContext.AuthActorType != string(types.ActorTypeService) || callContext.AuthActorID == nil || *callContext.AuthActorID != serviceID {
		t.Fatalf("reconstructed actor = %v/%v", callContext.AuthActorType, callContext.AuthActorID)
	}
}

func TestReconstructCallContextRejectsInvalidAuthentication(t *testing.T) {
	if _, err := reconstructCallContext(nil, 1, 2, "inbound", "call", "context", "sip:from@example.com", "sip:to@example.com"); err == nil {
		t.Fatal("reconstructCallContext() succeeded without authentication")
	}
}
