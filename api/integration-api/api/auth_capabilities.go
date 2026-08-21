package integration_api

import "github.com/rapidaai/pkg/types"

func requireProjectContext(auth types.AuthenticationPrinciple) (types.ProjectContext, error) {
	return types.RequireProject(auth)
}
