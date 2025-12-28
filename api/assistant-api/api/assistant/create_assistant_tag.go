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

// CreateAssistantTag implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistantTag(ctx context.Context, eRequest *assistant_api.CreateAssistantTagRequest) (*assistant_api.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantResponse](
			errors.New("unauthenticated request for CreateAssistantProviderModel"),
			"Please provider valid service credentials to create assistant tag, read docs @ docs.rapida.ai",
		)
	}
	_, err := assistantApi.assistantService.CreateOrUpdateAssistantTag(ctx, iAuth, eRequest.GetAssistantId(), eRequest.GetTags())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantResponse](
			err,
			"Unable to create tags for assistant, please try again in sometime.",
		)

	}
	assistant, err := assistantApi.assistantService.Get(ctx,
		iAuth,
		eRequest.GetAssistantId(),
		nil,
		internal_services.NewDefaultGetAssistantOption())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantResponse](
			err,
			"Unable to create tags for assistant, please try again in sometime.",
		)
	}
	out := &assistant_api.Assistant{}
	err = utils.Cast(assistant, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}
	return utils.Success[assistant_api.GetAssistantResponse, *assistant_api.Assistant](out)

}
