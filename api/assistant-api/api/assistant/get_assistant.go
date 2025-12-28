// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) GetAssistant(ctx context.Context, cepm *assistant_api.GetAssistantRequest) (*assistant_api.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantResponse](
			errors.New("unauthenticated request for get assistant"),
			"Please provider valid service credentials to perform GetAssistant, read docs @ docs.rapida.ai",
		)
	}

	ep, err := assistantApi.assistantService.Get(
		ctx,
		iAuth,
		cepm.
			GetAssistantDefinition().
			GetAssistantId(),
		utils.GetVersionDefinition(cepm.GetAssistantDefinition().GetVersion()),
		internal_services.NewDefaultGetAssistantOption())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantResponse](
			err,
			"Unable to get the assistant for given assistant id.",
		)
	}

	out := &assistant_api.Assistant{}
	err = utils.Cast(ep, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant %v", err)
	}
	return &assistant_api.GetAssistantResponse{
		Data:    out,
		Success: true,
		Code:    200,
	}, nil
}
