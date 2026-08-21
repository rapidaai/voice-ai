package internal_assistant_service

import "github.com/rapidaai/pkg/types"

type mutationContext struct {
	UserID  uint64
	Project types.ProjectContext
}

func requireUser(auth types.SimplePrinciple) (uint64, error) {
	return types.RequireUser(auth)
}

func requireProject(auth types.SimplePrinciple) (types.ProjectContext, error) {
	return types.RequireProject(auth)
}

func requireMutationContext(auth types.SimplePrinciple) (mutationContext, error) {
	userID, err := types.RequireUser(auth)
	if err != nil {
		return mutationContext{}, err
	}
	projectContext, err := types.RequireProject(auth)
	if err != nil {
		return mutationContext{}, err
	}
	return mutationContext{UserID: userID, Project: projectContext}, nil
}
