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

	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// GetAllAssistant implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantTool(ctx context.Context, cepm *assistant_api.GetAllAssistantToolRequest) (*assistant_api.GetAllAssistantToolResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	cnt, assistants, err := assistantApi.assistantToolService.GetAll(ctx, iAuth,
		cepm.GetAssistantId(),
		cepm.GetCriterias(),
		cepm.GetPaginate(),
	)
	if err != nil {
		return utils.Error[assistant_api.GetAllAssistantToolResponse](
			err,
			"Unable to get all the skill request.",
		)
	}
	out := []*assistant_api.AssistantTool{}
	err = utils.Cast(assistants, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant skill %v", err)
	}

	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[assistant_api.GetAllAssistantToolResponse, []*assistant_api.AssistantTool](
		totalItem,
		cepm.GetPaginate().GetPage(),
		out)
}
