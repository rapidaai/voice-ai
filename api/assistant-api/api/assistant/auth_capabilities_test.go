package assistant_api

import (
	"testing"

	"github.com/rapidaai/pkg/types"
)

type assistantCapabilityPrinciple struct {
	userID         uint64
	organizationID uint64
	projectID      uint64
}

func (p assistantCapabilityPrinciple) IsAuthenticated() bool   { return true }
func (p assistantCapabilityPrinciple) GetCurrentToken() string { return "" }
func (p assistantCapabilityPrinciple) Type() types.AuthType    { return types.AuthTypeUser }
func (p assistantCapabilityPrinciple) UserIdentity() (uint64, bool) {
	return p.userID, p.userID != 0
}
func (p assistantCapabilityPrinciple) OrganizationContext() (uint64, bool) {
	return p.organizationID, p.organizationID != 0
}
func (p assistantCapabilityPrinciple) ProjectContext() (types.ProjectContext, bool) {
	return types.ProjectContext{OrganizationID: p.organizationID, ProjectID: p.projectID}, p.organizationID != 0 && p.projectID != 0
}

func TestAssistantCapabilityPredicates(t *testing.T) {
	valid := assistantCapabilityPrinciple{userID: 1, organizationID: 2, projectID: 3}
	if !hasUserProjectCapability(valid) || !hasOrganizationCapability(valid) {
		t.Fatal("expected complete user project capability")
	}
	if hasUserProjectCapability(assistantCapabilityPrinciple{organizationID: 2, projectID: 3}) {
		t.Fatal("missing user identity must fail closed")
	}
}
