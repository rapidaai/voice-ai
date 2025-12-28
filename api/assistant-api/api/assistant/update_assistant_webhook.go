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

func (assistantApi *assistantGrpcApi) UpdateAssistantWebhook(ctx context.Context, cawr *protos.UpdateAssistantWebhookRequest) (*protos.GetAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[protos.GetAssistantWebhookResponse]()
	}
	wl, err := assistantApi.assistantWebhookService.Update(
		ctx,
		iAuth,
		cawr.GetAssistantId(),
		cawr.GetId(),
		cawr.GetAssistantEvents(),
		cawr.GetTimeoutSecond(),
		cawr.GetHttpMethod(),
		cawr.GetHttpUrl(),
		cawr.GetHttpHeaders(),
		cawr.GetHttpBody(),
		cawr.GetRetryStatusCodes(),
		cawr.GetMaxRetryCount(),
		cawr.GetExecutionPriority(),
		&cawr.Description)
	if err != nil {
		return exceptions.BadRequestError[protos.GetAssistantWebhookResponse]("Unable to create assistant webhook.")
	}
	aWebhook := &protos.AssistantWebhook{}
	err = utils.Cast(wl, aWebhook)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant webhook to the response object")
	}
	return utils.Success[protos.GetAssistantWebhookResponse, *protos.AssistantWebhook](aWebhook)
}
