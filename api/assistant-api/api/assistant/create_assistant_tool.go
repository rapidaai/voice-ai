// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) CreateAssistantTool(ctx context.Context, atr *assistant_api.CreateAssistantToolRequest) (*assistant_api.GetAssistantToolResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantToolResponse](
			errors.New("unauthenticated request for get assistant"),
			"Please provider valid service credentials to perform CreateAssistantTool, read docs @ docs.rapida.ai",
		)
	}

	aT, err := assistantApi.
		assistantToolService.
		Create(
			ctx,
			iAuth,
			atr.GetAssistantId(),
			atr.GetName(),
			atr.GetDescription(),
			atr.GetFields().AsMap(),
			atr.GetExecutionMethod(),
			atr.GetExecutionOptions(),
		)

	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantToolResponse](err.Error())
	}

	out := &assistant_api.AssistantTool{}
	err = utils.Cast(aT, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}
	return utils.Success[assistant_api.GetAssistantToolResponse, *assistant_api.AssistantTool](out)
}

func (assistantApi *assistantGrpcApi) createAssistantTool(ctx context.Context,
	iAuth types.SimplePrinciple,
	assistantId uint64,
	atr *assistant_api.CreateAssistantToolRequest) (*internal_assistant_entity.AssistantTool, error) {
	return assistantApi.
		assistantToolService.
		Create(
			ctx,
			iAuth,
			assistantId,
			atr.GetName(),
			atr.GetDescription(),
			atr.GetFields().AsMap(),
			atr.GetExecutionMethod(),
			atr.GetExecutionOptions(),
		)

}
