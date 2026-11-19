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

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) UpdateAssistantTool(ctx context.Context, cawr *protos.UpdateAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	wl, err := assistantApi.assistantToolService.Update(
		ctx,
		iAuth,
		cawr.GetId(),
		cawr.GetAssistantId(),
		cawr.GetName(),
		&cawr.Description,
		cawr.GetFields().AsMap(),
		cawr.GetExecutionMethod(),
		cawr.GetExecutionOptions())
	if err != nil {
		return exceptions.BadRequestError[protos.GetAssistantToolResponse](err.Error())
	}
	aAnalysis := &protos.AssistantTool{}
	err = utils.Cast(wl, aAnalysis)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant tool to the response object")
	}
	return utils.Success[protos.GetAssistantToolResponse, *protos.AssistantTool](aAnalysis)
}
