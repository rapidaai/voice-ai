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
	assistant_api "github.com/rapidaai/protos"
)

// CreateAssistantWebhook implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistantWebhook(ctx context.Context, cawr *assistant_api.CreateAssistantWebhookRequest) (*assistant_api.GetAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAssistantWebhookResponse]()
	}
	wl, err := assistantApi.assistantWebhookService.Create(
		ctx,
		iAuth,
		cawr.GetAssistantId(),
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
		return exceptions.BadRequestError[assistant_api.GetAssistantWebhookResponse]("Unable to create assistant webhook.")
	}
	aWebhook := &assistant_api.AssistantWebhook{}
	err = utils.Cast(wl, aWebhook)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant webhook to the response object")
	}
	return utils.Success[assistant_api.GetAssistantWebhookResponse, *assistant_api.AssistantWebhook](aWebhook)
}
