package observability_api

import "github.com/rapidaai/pkg/types"

func requireTelemetryProjectContext(auth types.AuthenticationPrinciple) (types.ProjectContext, error) {
	return types.RequireProject(auth)
}
