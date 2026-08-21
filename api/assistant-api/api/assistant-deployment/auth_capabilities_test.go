package assistant_deployment_api

import (
	"testing"

	"github.com/rapidaai/pkg/types"
)

type deploymentCapabilityPrinciple struct {
	userID uint64
}

func (p deploymentCapabilityPrinciple) IsAuthenticated() bool   { return true }
func (p deploymentCapabilityPrinciple) GetCurrentToken() string { return "" }
func (p deploymentCapabilityPrinciple) Type() types.AuthType    { return types.AuthTypeUser }
func (p deploymentCapabilityPrinciple) UserIdentity() (uint64, bool) {
	return p.userID, p.userID != 0
}
func (deploymentCapabilityPrinciple) ProjectContext() (types.ProjectContext, bool) {
	return types.ProjectContext{OrganizationID: 2, ProjectID: 3}, true
}

func TestDeploymentCapabilityPredicates(t *testing.T) {
	if !hasUserProjectCapability(deploymentCapabilityPrinciple{userID: 1}) {
		t.Fatal("expected complete user project capability")
	}
	if hasUserProjectCapability(deploymentCapabilityPrinciple{}) {
		t.Fatal("missing user identity must fail closed")
	}
}
