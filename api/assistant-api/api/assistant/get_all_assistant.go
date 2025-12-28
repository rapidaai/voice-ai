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
	"github.com/rapidaai/protos"
)

// GetAllAssistant implements protos.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistant(ctx context.Context, cepm *protos.GetAllAssistantRequest) (*protos.GetAllAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAllassistant")
		return utils.Error[protos.GetAllAssistantResponse](
			errors.New("unauthenticated request for get allassistant"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
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

	return utils.PaginatedSuccess[protos.GetAllAssistantResponse, []*protos.Assistant](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}
