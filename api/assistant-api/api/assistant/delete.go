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
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) DeleteAssistant(ctx context.Context, cer *assistant_api.DeleteAssistantRequest) (*assistant_api.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for UpdateAssistantDetail")
		return utils.Error[assistant_api.GetAssistantResponse](
			errors.New("unauthenticated request for UpdateAssistantDetail"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	assistant, err := assistantApi.assistantService.DeleteAssistant(ctx,
		iAuth,
		cer.GetId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantResponse](
			err,
			"Unable to update assistant, please try again in sometime",
		)
	}
	out := &assistant_api.Assistant{}
	err = utils.Cast(assistant, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant provider model to the response object")
	}
	return utils.Success[assistant_api.GetAssistantResponse, *assistant_api.Assistant](out)

}

func (assistantApi *assistantGrpcApi) DeleteAssistantAnalysis(ctx context.Context, cer *assistant_api.DeleteAssistantAnalysisRequest) (*assistant_api.GetAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for DeleteAssistantAnalysisRequest")
		return utils.Error[assistant_api.GetAssistantAnalysisResponse](
			errors.New("unauthenticated request for DeleteAssistantAnalysisRequest"),
			"Please provider valid service credentials to perfom DeleteAssistantAnalysisRequest, read docs @ docs.rapida.ai",
		)
	}
	analysis, err := assistantApi.assistantAnalysisService.Delete(ctx,
		iAuth,
		cer.GetId(), cer.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantAnalysisResponse](
			err,
			"Unable to update assistant analysis, please try again in sometime",
		)
	}
	out := &assistant_api.AssistantAnalysis{}
	err = utils.Cast(analysis, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant analysis to the response object")
	}
	return utils.Success[assistant_api.GetAssistantAnalysisResponse, *assistant_api.AssistantAnalysis](out)

}

func (assistantApi *assistantGrpcApi) DeleteAssistantWebhook(ctx context.Context, cer *assistant_api.DeleteAssistantWebhookRequest) (*assistant_api.GetAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for DeleteAssistantWebhookRequest")
		return utils.Error[assistant_api.GetAssistantWebhookResponse](
			errors.New("unauthenticated request for DeleteAssistantWebhookRequest"),
			"Please provider valid service credentials to perfom DeleteAssistantWebhookRequest, read docs @ docs.rapida.ai",
		)
	}
	analysis, err := assistantApi.assistantWebhookService.Delete(ctx,
		iAuth,
		cer.GetId(), cer.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantWebhookResponse](
			err,
			"Unable to update assistant analysis, please try again in sometime",
		)
	}
	out := &assistant_api.AssistantWebhook{}
	err = utils.Cast(analysis, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant analysis to the response object")
	}
	return utils.Success[assistant_api.GetAssistantWebhookResponse, *assistant_api.AssistantWebhook](out)

}

func (assistantApi *assistantGrpcApi) DeleteAssistantTool(ctx context.Context, cer *assistant_api.DeleteAssistantToolRequest) (*assistant_api.GetAssistantToolResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for DeleteAssistantToolRequest")
		return utils.Error[assistant_api.GetAssistantToolResponse](
			errors.New("unauthenticated request for DeleteAssistantToolRequest"),
			"Please provider valid service credentials to perfom DeleteAssistantToolRequest, read docs @ docs.rapida.ai",
		)
	}
	analysis, err := assistantApi.assistantToolService.Delete(ctx,
		iAuth,
		cer.GetId(),
		cer.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantToolResponse](
			err,
			"Unable to update assistant analysis, please try again in sometime",
		)
	}
	out := &assistant_api.AssistantTool{}
	err = utils.Cast(analysis, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant analysis to the response object")
	}
	return utils.Success[assistant_api.GetAssistantToolResponse, *assistant_api.AssistantTool](out)

}

func (assistantApi *assistantGrpcApi) DeleteAssistantKnowledge(ctx context.Context, cer *assistant_api.DeleteAssistantKnowledgeRequest) (*assistant_api.GetAssistantKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for DeleteAssistantKnowledgeRequest")
		return utils.Error[assistant_api.GetAssistantKnowledgeResponse](
			errors.New("unauthenticated request for DeleteAssistantKnowledgeRequest"),
			"Please provider valid service credentials to perfom DeleteAssistantKnowledgeRequest, read docs @ docs.rapida.ai",
		)
	}
	analysis, err := assistantApi.assistantKnowledgeService.Delete(ctx,
		iAuth,
		cer.GetId(),
		cer.GetAssistantId(),
	)
	if err != nil {
		return utils.Error[assistant_api.GetAssistantKnowledgeResponse](
			err,
			"Unable to update assistant analysis, please try again in sometime",
		)
	}
	out := &assistant_api.AssistantKnowledge{}
	err = utils.Cast(analysis, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant analysis to the response object")
	}
	return utils.Success[assistant_api.GetAssistantKnowledgeResponse, *assistant_api.AssistantKnowledge](out)

}
