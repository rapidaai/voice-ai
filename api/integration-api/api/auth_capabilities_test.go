package integration_api

import (
	"testing"

	"github.com/rapidaai/pkg/types"
)

type integrationCapabilityPrinciple struct {
	projectID uint64
}

func (p integrationCapabilityPrinciple) IsAuthenticated() bool   { return true }
func (p integrationCapabilityPrinciple) GetCurrentToken() string { return "" }
func (p integrationCapabilityPrinciple) Type() types.AuthType    { return types.AuthTypeProject }
func (p integrationCapabilityPrinciple) ProjectContext() (types.ProjectContext, bool) {
	return types.ProjectContext{OrganizationID: 2, ProjectID: p.projectID}, p.projectID != 0
}

func TestRequireProjectContext(t *testing.T) {
	context, err := requireProjectContext(integrationCapabilityPrinciple{projectID: 3})
	if err != nil || context.OrganizationID != 2 || context.ProjectID != 3 {
		t.Fatalf("requireProjectContext() = %+v, %v", context, err)
	}
	if _, err := requireProjectContext(integrationCapabilityPrinciple{}); err == nil {
		t.Fatal("requireProjectContext() error = nil, want missing project error")
	}
}
