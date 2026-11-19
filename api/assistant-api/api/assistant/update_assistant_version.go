// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) UpdateAssistantVersion(ctx context.Context, cer *protos.UpdateAssistantVersionRequest) (*protos.GetAssistantResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	ep, err := assistantApi.assistantService.UpdateAssistantVersion(
		ctx,
		iAuth,
		cer.GetAssistantId(),
		enums.ToAssistantProvider(cer.GetAssistantProvider()),
		cer.GetAssistantProviderId())
	if err != nil {
		return utils.Error[protos.GetAssistantResponse](
			errors.New("unauthenticated request for updateassistantversion"),
			"Unable to update assistant for given assistant id.",
		)
	}
	out := &protos.Assistant{}
	err = utils.Cast(ep, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.Success[protos.GetAssistantResponse, *protos.Assistant](out)

}
