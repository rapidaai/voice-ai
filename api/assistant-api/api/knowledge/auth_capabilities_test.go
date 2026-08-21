package knowledge_api

import (
	"testing"

	"github.com/rapidaai/pkg/types"
)

type knowledgeCapabilityPrinciple struct {
	projectID uint64
}

func (p knowledgeCapabilityPrinciple) IsAuthenticated() bool   { return true }
func (p knowledgeCapabilityPrinciple) GetCurrentToken() string { return "" }
func (p knowledgeCapabilityPrinciple) Type() types.AuthType    { return types.AuthTypeProject }
func (p knowledgeCapabilityPrinciple) ProjectContext() (types.ProjectContext, bool) {
	return types.ProjectContext{OrganizationID: 2, ProjectID: p.projectID}, p.projectID != 0
}

func TestKnowledgeProjectCapability(t *testing.T) {
	if !hasProjectCapability(knowledgeCapabilityPrinciple{projectID: 3}) {
		t.Fatal("expected project capability")
	}
	if hasProjectCapability(knowledgeCapabilityPrinciple{}) {
		t.Fatal("missing project identity must fail closed")
	}
}
