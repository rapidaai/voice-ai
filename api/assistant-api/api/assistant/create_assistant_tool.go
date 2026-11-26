// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) CreateAssistantTool(ctx context.Context, atr *assistant_api.CreateAssistantToolRequest) (*assistant_api.GetAssistantToolResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	aT, err := assistantApi.
		assistantToolService.
		Create(
			ctx,
			iAuth,
			atr.GetAssistantId(),
			atr.GetName(),
			&atr.Description,
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
	iAuth authentication.Principal,
	assistantId uint64,
	atr *assistant_api.CreateAssistantToolRequest) (*internal_assistant_entity.AssistantTool, error) {
	return assistantApi.
		assistantToolService.
		Create(
			ctx,
			iAuth,
			assistantId,
			atr.GetName(),
			&atr.Description,
			atr.GetFields().AsMap(),
			atr.GetExecutionMethod(),
			atr.GetExecutionOptions(),
		)

}
