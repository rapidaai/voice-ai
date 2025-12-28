// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// GetAssistantWebhook implements protos.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAssistantWebhook(ctx context.Context, gawr *protos.GetAssistantWebhookRequest) (*protos.GetAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[protos.GetAssistantWebhookResponse]()
	}
	tlp, err := assistantApi.assistantWebhookService.Get(ctx, iAuth, gawr.GetId(), gawr.GetAssistantId())
	if err != nil {
		return utils.Error[protos.GetAssistantWebhookResponse](
			err,
			"Unable to get the webhook for given webhook id.",
		)
	}
	out := &protos.AssistantWebhook{}
	err = utils.Cast(tlp, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant analysis %v", err)
	}

	return utils.Success[protos.GetAssistantWebhookResponse, *protos.AssistantWebhook](out)
}
