package observability_api

import (
	"testing"

	"github.com/rapidaai/pkg/types"
)

func TestRequireTelemetryProjectContextFromServiceScope(t *testing.T) {
	organizationID := uint64(7)
	projectID := uint64(8)
	context, err := requireTelemetryProjectContext(&types.ServiceScope{OrganizationId: &organizationID, ProjectId: &projectID})
	if err != nil || context.OrganizationID != organizationID || context.ProjectID != projectID {
		t.Fatalf("requireTelemetryProjectContext() = %+v, %v", context, err)
	}
}
