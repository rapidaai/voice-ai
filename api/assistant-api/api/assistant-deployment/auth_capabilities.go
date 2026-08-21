package assistant_deployment_api

import "github.com/rapidaai/pkg/types"

func hasProjectCapability(auth types.AuthenticationPrinciple) bool {
	_, err := types.RequireProject(auth)
	return err == nil
}

func hasUserProjectCapability(auth types.AuthenticationPrinciple) bool {
	if _, err := types.RequireUser(auth); err != nil {
		return false
	}
	return hasProjectCapability(auth)
}
