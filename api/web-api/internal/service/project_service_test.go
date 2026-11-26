// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_service

import (
	"context"
	"testing"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
)

type projectServiceCredentialContractStub struct {
	ProjectService
	receivedRoleDefinitionID uint64
}

func (stub *projectServiceCredentialContractStub) CreateCredential(_ context.Context, _ authentication.Principal, _ string, _, _, roleDefinitionID uint64) (*internal_entity.ProjectCredential, error) {
	stub.receivedRoleDefinitionID = roleDefinitionID
	return &internal_entity.ProjectCredential{RoleDefinitionID: roleDefinitionID}, nil
}

func TestProjectServiceCreateCredentialRequiresRoleDefinitionID(t *testing.T) {
	stub := &projectServiceCredentialContractStub{}
	var service ProjectService = stub
	credential, err := service.CreateCredential(context.Background(), authentication.Principal{}, "publishable key", 100, 9, 117)
	if err != nil {
		t.Fatalf("CreateCredential returned error: %v", err)
	}
	if stub.receivedRoleDefinitionID != 117 || credential.RoleDefinitionID != 117 {
		t.Fatalf("role definition ID was not forwarded: stub=%d credential=%d", stub.receivedRoleDefinitionID, credential.RoleDefinitionID)
	}
}
