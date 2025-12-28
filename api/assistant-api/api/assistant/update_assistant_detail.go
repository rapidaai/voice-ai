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

func (assistantApi *assistantGrpcApi) UpdateAssistantDetail(ctx context.Context, cer *protos.UpdateAssistantDetailRequest) (*protos.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for UpdateAssistantDetail")
		return utils.Error[protos.GetAssistantResponse](
			errors.New("unauthenticated request for UpdateAssistantDetail"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	_, err := assistantApi.assistantService.UpdateAssistantDetail(ctx,
		iAuth,
		cer.GetAssistantId(), cer.GetName(), cer.GetDescription())
	if err != nil {
		return utils.Error[protos.GetAssistantResponse](
			err,
			"Unable to update assistant, please try again in sometime",
		)
	}
	assistant, err := assistantApi.assistantService.Get(ctx, iAuth, cer.GetAssistantId(), nil, internal_services.NewDefaultGetAssistantOption())
	if err != nil {
		return utils.Error[protos.GetAssistantResponse](
			err,
			"Unable to get the assistant for given assistant id.",
		)
	}

	out := &protos.Assistant{}
	err = utils.Cast(assistant, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}
	return utils.Success[protos.GetAssistantResponse, *protos.Assistant](out)

}
