package knowledge_api

import "github.com/rapidaai/pkg/types"

func hasProjectCapability(auth types.AuthenticationPrinciple) bool {
	_, err := types.RequireProject(auth)
	return err == nil
}
