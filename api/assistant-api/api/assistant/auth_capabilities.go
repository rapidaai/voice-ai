package assistant_api

import "github.com/rapidaai/pkg/types"

func hasUserCapability(auth types.AuthenticationPrinciple) bool {
	_, err := types.RequireUser(auth)
	return err == nil
}

func hasOrganizationCapability(auth types.AuthenticationPrinciple) bool {
	_, err := types.RequireOrganization(auth)
	return err == nil
}

func hasProjectCapability(auth types.AuthenticationPrinciple) bool {
	_, err := types.RequireProject(auth)
	return err == nil
}

func hasUserProjectCapability(auth types.AuthenticationPrinciple) bool {
	return hasUserCapability(auth) && hasProjectCapability(auth)
}
