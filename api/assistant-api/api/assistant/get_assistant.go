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
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) GetAssistant(ctx context.Context, cepm *assistant_api.GetAssistantRequest) (*assistant_api.GetAssistantResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
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
