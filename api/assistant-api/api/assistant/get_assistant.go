// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) GetAssistant(ctx context.Context, cepm *assistant_api.GetAssistantRequest) (*assistant_api.GetAssistantResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	agent, err := assistantApi.assistantService.Get(
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
	err = utils.Cast(agent, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant %v", err)
	}
	return utils.Success[protos.GetAssistantResponse, *protos.Assistant](out)

}
