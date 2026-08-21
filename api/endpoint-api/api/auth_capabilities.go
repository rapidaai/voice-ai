package endpoint_api

import (
	"context"

	"github.com/rapidaai/pkg/types"
)

func getProjectPrincipleGRPC(ctx context.Context) (types.SimplePrinciple, bool) {
	auth, authenticated := types.GetSimplePrincipleGRPC(ctx)
	if !authenticated {
		return nil, false
	}
	if _, err := types.RequireProject(auth); err != nil {
		return nil, false
	}
	return auth, true
}
