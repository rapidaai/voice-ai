// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) UpdateAssistantVersion(ctx context.Context, cer *protos.UpdateAssistantVersionRequest) (*protos.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for UpdateassistantVersion")
		return utils.Error[protos.GetAssistantResponse](
			errors.New("unauthenticated request for updateassistantversion"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	assistantApi.logger.Debug("check %+v and got %+v", cer)
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
