package internal_knowledge_service

import "github.com/rapidaai/pkg/types"

func requireMutationContext(auth types.SimplePrinciple) (uint64, types.ProjectContext, error) {
	userID, err := types.RequireUser(auth)
	if err != nil {
		return 0, types.ProjectContext{}, err
	}
	projectContext, err := types.RequireProject(auth)
	if err != nil {
		return 0, types.ProjectContext{}, err
	}
	return userID, projectContext, nil
}
