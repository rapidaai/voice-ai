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

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// GetAllAssistant implements protos.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistant(ctx context.Context, cepm *protos.GetAllAssistantRequest) (*protos.GetAllAssistantResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	cnt, assistants, err := assistantApi.assistantService.GetAll(ctx, iAuth,
		cepm.GetCriterias(),
		cepm.GetPaginate(), internal_services.NewDefaultGetAssistantOption())
	if err != nil {
		return utils.Error[protos.GetAllAssistantResponse](
			err,
			"Unable to get all the assistant.",
		)
	}
	out := []*protos.Assistant{}
	err = utils.Cast(assistants, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[protos.GetAllAssistantResponse, []*protos.Assistant](
		totalItem,
		cepm.GetPaginate().GetPage(),
		out)
}
