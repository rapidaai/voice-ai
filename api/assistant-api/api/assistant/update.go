package assistant_api

import (
	"context"
	"errors"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/exceptions"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) UpdateAssistantVersion(ctx context.Context, cer *assistant_api.UpdateAssistantVersionRequest) (*assistant_api.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for UpdateassistantVersion")
		return utils.Error[assistant_api.GetAssistantResponse](
			errors.New("unauthenticated request for updateassistantversion"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	assistantApi.logger.Debug("check %+v and got %+v", cer)
	ep, err := assistantApi.assistantService.UpdateAssistantVersion(
		ctx,
		iAuth,
		cer.GetAssistantId(),
		type_enums.ToAssistantProvider(cer.GetAssistantProvider()),
		cer.GetAssistantProviderId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantResponse](
			errors.New("unauthenticated request for updateassistantversion"),
			"Unable to update assistant for given assistant id.",
		)
	}
	out := &assistant_api.Assistant{}
	err = utils.Cast(ep, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.Success[assistant_api.GetAssistantResponse, *assistant_api.Assistant](out)

}

func (assistantApi *assistantGrpcApi) UpdateAssistantDetail(ctx context.Context, cer *assistant_api.UpdateAssistantDetailRequest) (*assistant_api.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for UpdateAssistantDetail")
		return utils.Error[assistant_api.GetAssistantResponse](
			errors.New("unauthenticated request for UpdateAssistantDetail"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	_, err := assistantApi.assistantService.UpdateAssistantDetail(ctx,
		iAuth,
		cer.GetAssistantId(), cer.GetName(), cer.GetDescription())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantResponse](
			err,
			"Unable to update assistant, please try again in sometime",
		)
	}
	assistant, err := assistantApi.assistantService.Get(ctx, iAuth, cer.GetAssistantId(), nil, internal_services.NewDefaultGetAssistantOption())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantResponse](
			err,
			"Unable to get the assistant for given assistant id.",
		)
	}

	out := &assistant_api.Assistant{}
	err = utils.Cast(assistant, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}
	return utils.Success[assistant_api.GetAssistantResponse, *assistant_api.Assistant](out)

}

func (assistantApi *assistantGrpcApi) UpdateAssistantWebhook(ctx context.Context, cawr *assistant_api.UpdateAssistantWebhookRequest) (*assistant_api.GetAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAssistantWebhookResponse]()
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
		return exceptions.BadRequestError[assistant_api.GetAssistantWebhookResponse]("Unable to create assistant webhook.")
	}
	aWebhook := &assistant_api.AssistantWebhook{}
	err = utils.Cast(wl, aWebhook)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant webhook to the response object")
	}
	return utils.Success[assistant_api.GetAssistantWebhookResponse, *assistant_api.AssistantWebhook](aWebhook)
}

func (assistantApi *assistantGrpcApi) UpdateAssistantAnalysis(ctx context.Context, cawr *assistant_api.UpdateAssistantAnalysisRequest) (*assistant_api.GetAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for UpdateAssistantAnalysis")
		return exceptions.AuthenticationError[assistant_api.GetAssistantAnalysisResponse]()
	}
	wl, err := assistantApi.assistantAnalysisService.Update(
		ctx,
		iAuth,
		cawr.GetAssistantId(),
		cawr.GetId(),
		cawr.GetName(),
		cawr.GetEndpointId(),
		cawr.GetEndpointVersion(),
		cawr.GetEndpointParameters(),
		cawr.GetExecutionPriority(),
		&cawr.Description)
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantAnalysisResponse]("Unable to create assistant webhook.")
	}
	aAnalysis := &assistant_api.AssistantAnalysis{}
	err = utils.Cast(wl, aAnalysis)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant analysis to the response object")
	}
	return utils.Success[assistant_api.GetAssistantAnalysisResponse, *assistant_api.AssistantAnalysis](aAnalysis)
}

func (assistantApi *assistantGrpcApi) UpdateAssistantTool(ctx context.Context, cawr *assistant_api.UpdateAssistantToolRequest) (*assistant_api.GetAssistantToolResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for UpdateAssistantTool")
		return exceptions.AuthenticationError[assistant_api.GetAssistantToolResponse]()
	}

	wl, err := assistantApi.assistantToolService.Update(
		ctx,
		iAuth,
		cawr.GetId(),
		cawr.GetAssistantId(),
		cawr.GetName(),
		cawr.GetDescription(),
		cawr.GetFields().AsMap(),
		cawr.GetExecutionMethod(),
		cawr.GetExecutionOptions())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantToolResponse](err.Error())
	}
	aAnalysis := &assistant_api.AssistantTool{}
	err = utils.Cast(wl, aAnalysis)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant tool to the response object")
	}
	return utils.Success[assistant_api.GetAssistantToolResponse, *assistant_api.AssistantTool](aAnalysis)
}

func (assistantApi *assistantGrpcApi) UpdateAssistantKnowledge(ctx context.Context, cawr *assistant_api.UpdateAssistantKnowledgeRequest) (*assistant_api.GetAssistantKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for UpdateAssistantKnowledge")
		return exceptions.AuthenticationError[assistant_api.GetAssistantKnowledgeResponse]()
	}
	wl, err := assistantApi.assistantKnowledgeService.Update(
		ctx,
		iAuth,
		cawr.GetId(),
		cawr.GetAssistantId(),
		cawr.GetKnowledgeId(),
		gorm_types.RetrievalMethod(cawr.GetRetrievalMethod()),
		cawr.GetRerankerEnable(),
		cawr.GetScoreThreshold(),
		cawr.GetTopK(),
		&cawr.RerankerModelProviderId,
		&cawr.RerankerModelProviderName,
		cawr.GetAssistantKnowledgeRerankerOptions())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantKnowledgeResponse](err.Error())
	}
	aAnalysis := &assistant_api.AssistantKnowledge{}
	err = utils.Cast(wl, aAnalysis)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant knowledge to the response object")
	}
	return utils.Success[assistant_api.GetAssistantKnowledgeResponse, *assistant_api.AssistantKnowledge](aAnalysis)
}
