package assistant_api

import (
	"context"
	"errors"
	"fmt"

	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/exceptions"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/encoding/protojson"
)

// CreateAssistant implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistant(ctx context.Context, cer *assistant_api.CreateAssistantRequest) (*assistant_api.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	// creating assistant
	assistant, err := assistantApi.
		assistantService.
		CreateAssistant(
			ctx,
			iAuth,
			cer.GetName(),
			cer.GetDescription(),
			cer.GetVisibility(),
			cer.GetSource(),
			&cer.SourceIdentifier,
			cer.GetLanguage(),
		)
	if err != nil {
		return utils.Error[assistant_api.GetAssistantResponse](
			err,
			"Unable to create endpoint, please try again later",
		)
	}

	prd := cer.GetAssistantProvider().GetAssistantProvider()
	switch provider := prd.(type) {
	case *assistant_api.CreateAssistantProviderRequest_Model:
		providerModel, err := assistantApi.assistantService.CreateAssistantProviderModel(
			ctx,
			iAuth,
			assistant.Id,
			cer.GetAssistantProvider().GetDescription(),
			protojson.Format(provider.Model.GetTemplate()),
			provider.Model.GetModelProviderId(),
			provider.Model.GetModelProviderName(),
			provider.Model.GetAssistantModelOptions(),
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantResponse](
				err,
				"Unable to create assistant provider model, please try again later",
			)
		}
		_, err = assistantApi.
			assistantService.AttachProviderModelToAssistant(
			ctx,
			iAuth,
			assistant.Id,
			type_enums.MODEL,
			providerModel.Id,
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantResponse](
				err,
				"Unable to attach assistant provider model, please try again later",
			)
		}

	case *assistant_api.CreateAssistantProviderRequest_Agentkit:
		agentKitProvider, err := assistantApi.assistantService.CreateAssistantProviderAgentkit(
			ctx,
			iAuth,
			assistant.Id,
			cer.GetAssistantProvider().GetDescription(),
			provider.Agentkit.GetAgentKitUrl(),
			provider.Agentkit.GetCertificate(),
			provider.Agentkit.GetMetadata(),
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantResponse](
				err,
				"Unable to create assistant provider model, please check the argument and try again.",
			)
		}
		_, err = assistantApi.
			assistantService.AttachProviderModelToAssistant(
			ctx,
			iAuth,
			assistant.Id,
			type_enums.AGENTKIT,
			agentKitProvider.Id,
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantResponse](
				err,
				"Unable to attach assistant provider agentkit, please try again later",
			)
		}

	case *assistant_api.CreateAssistantProviderRequest_Websocket:
		websocketProvider, err := assistantApi.assistantService.CreateAssistantProviderWebsocket(
			ctx,
			iAuth,
			assistant.Id,
			cer.GetAssistantProvider().GetDescription(),
			provider.Websocket.GetWebsocketUrl(),
			provider.Websocket.GetHeaders(),
			provider.Websocket.GetConnectionParameters(),
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantResponse](
				err,
				"Unable to attach assistant provider agentkit, please try again later",
			)
		}
		_, err = assistantApi.
			assistantService.AttachProviderModelToAssistant(
			ctx,
			iAuth,
			assistant.Id,
			type_enums.WEBSOCKET,
			websocketProvider.Id,
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantResponse](
				err,
				"Unable to attach assistant provider agentkit, please try again later",
			)
		}

	}

	for _, tl := range cer.GetAssistantTools() {
		_, err := assistantApi.createAssistantTool(
			ctx,
			iAuth,
			assistant.Id,
			tl)
		if err != nil {
			assistantApi.logger.Errorf("Unable to create assistant tools, please try again later with error %+v", err)
		}
	}

	for _, ak := range cer.GetAssistantKnowledges() {
		_, err := assistantApi.createAssistantKnowledge(
			ctx,
			iAuth,
			assistant.Id,
			ak)
		if err != nil {
			assistantApi.logger.Errorf("Unable to create assistant knowledge, please try again later with error %+v", err)
		}
	}

	_, err = assistantApi.assistantService.CreateOrUpdateAssistantTag(ctx, iAuth, assistant.Id, cer.GetTags())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantResponse](
			err,
			"Unable to create assistant tags, please try again.",
		)
	}

	out := &assistant_api.Assistant{}
	err = utils.Cast(assistant, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant provider model to the response object")
	}
	return utils.Success[assistant_api.GetAssistantResponse, *assistant_api.Assistant](out)
}

// CreateAssistantProviderModel implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistantProvider(ctx context.Context,
	iRequest *assistant_api.CreateAssistantProviderRequest) (*assistant_api.GetAssistantProviderResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantProviderResponse](
			errors.New("unauthenticated request for GetAssistantProviderResponse"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	assistant, err := assistantApi.assistantService.Get(ctx,
		iAuth,
		iRequest.GetAssistantId(), nil, internal_services.NewDefaultGetAssistantOption())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantProviderResponse](
			err,
			"Unable to identify assistant version, please try again later",
		)
	}

	prd := iRequest.GetAssistantProvider()
	switch provider := prd.(type) {
	case *assistant_api.CreateAssistantProviderRequest_Model:
		providerModel, err := assistantApi.assistantService.CreateAssistantProviderModel(
			ctx,
			iAuth,
			assistant.Id,
			iRequest.GetDescription(),
			protojson.Format(provider.Model.GetTemplate()),
			provider.Model.GetModelProviderId(),
			provider.Model.GetModelProviderName(),
			provider.Model.GetAssistantModelOptions(),
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantProviderResponse](
				err,
				"Unable to create assistant provider model, please check the argument and try again.",
			)
		}
		aProviderModel := &assistant_api.AssistantProviderModel{}
		err = utils.Cast(providerModel, aProviderModel)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the assistant provider model to the response object")
		}
		return utils.Success[
			assistant_api.GetAssistantProviderResponse,
			*assistant_api.
				GetAssistantProviderResponse_AssistantProviderModel](
			&assistant_api.GetAssistantProviderResponse_AssistantProviderModel{
				AssistantProviderModel: aProviderModel,
			})
	case *assistant_api.CreateAssistantProviderRequest_Agentkit:
		agentKitProvider, err := assistantApi.assistantService.CreateAssistantProviderAgentkit(
			ctx,
			iAuth,
			assistant.Id,
			iRequest.GetDescription(),
			provider.Agentkit.GetAgentKitUrl(),
			provider.Agentkit.GetCertificate(),
			provider.Agentkit.GetMetadata(),
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantProviderResponse](
				err,
				"Unable to create assistant provider model, please check the argument and try again.",
			)
		}
		aProviderModel := &assistant_api.AssistantProviderAgentkit{}
		err = utils.Cast(agentKitProvider, aProviderModel)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the assistant provider model to the response object")
		}
		return utils.Success[
			assistant_api.GetAssistantProviderResponse,
			*assistant_api.
				GetAssistantProviderResponse_AssistantProviderAgentkit](
			&assistant_api.GetAssistantProviderResponse_AssistantProviderAgentkit{
				AssistantProviderAgentkit: aProviderModel,
			})
	case *assistant_api.CreateAssistantProviderRequest_Websocket:
		websocketProvider, err := assistantApi.assistantService.CreateAssistantProviderWebsocket(
			ctx,
			iAuth,
			assistant.Id,
			iRequest.GetDescription(),
			provider.Websocket.GetWebsocketUrl(),
			provider.Websocket.GetHeaders(),
			provider.Websocket.GetConnectionParameters(),
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantProviderResponse](
				err,
				"Unable to create assistant provider model, please check the argument and try again.",
			)
		}
		aProviderModel := &assistant_api.AssistantProviderWebsocket{}
		err = utils.Cast(websocketProvider, aProviderModel)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the assistant provider model to the response object")
		}
		return utils.Success[
			assistant_api.GetAssistantProviderResponse,
			*assistant_api.
				GetAssistantProviderResponse_AssistantProviderWebsocket](
			&assistant_api.GetAssistantProviderResponse_AssistantProviderWebsocket{
				AssistantProviderWebsocket: aProviderModel,
			})
	}
	return utils.Error[assistant_api.GetAssistantProviderResponse](
		fmt.Errorf("illegal request for creating new assistant provider"),
		"illegal request for creating new assistant provider",
	)
}

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

// CreateAssistantWebhook implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistantAnalysis(ctx context.Context, cawr *assistant_api.CreateAssistantAnalysisRequest) (*assistant_api.GetAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAssistantAnalysisResponse]()
	}
	wl, err := assistantApi.assistantAnalysisService.Create(
		ctx,
		iAuth,
		cawr.GetAssistantId(),
		cawr.GetName(),
		cawr.GetEndpointId(),
		cawr.GetEndpointVersion(),
		cawr.GetEndpointParameters(),
		cawr.GetExecutionPriority(),
		&cawr.Description)
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantAnalysisResponse]("Unable to create assistant analysis.")
	}
	aAnalysis := &assistant_api.AssistantAnalysis{}
	err = utils.Cast(wl, aAnalysis)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant analysis to the response object")
	}
	return utils.Success[assistant_api.GetAssistantAnalysisResponse, *assistant_api.AssistantAnalysis](aAnalysis)
}

// CreateAssistantKnowledgeConfiguration implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistantKnowledge(ctx context.Context, cepm *assistant_api.CreateAssistantKnowledgeRequest) (*assistant_api.GetAssistantKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantKnowledgeResponse](
			errors.New("unauthenticated request for get assistant"),
			"Please provider valid service credentials to perform CreateAssistantKnowledge, read docs @ docs.rapida.ai",
		)
	}
	aK, err := assistantApi.assistantKnowledgeService.Create(
		ctx,
		iAuth,
		cepm.GetAssistantId(),
		cepm.GetKnowledgeId(),
		gorm_types.RetrievalMethod(cepm.GetRetrievalMethod()),
		cepm.GetRerankerEnable(),
		cepm.GetScoreThreshold(),
		cepm.GetTopK(),
		&cepm.RerankerModelProviderId,
		&cepm.RerankerModelProviderName,
		cepm.GetAssistantKnowledgeRerankerOptions(),
	)
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantKnowledgeResponse](err.Error())
	}

	out := &assistant_api.AssistantKnowledge{}
	err = utils.Cast(aK, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant knowledge %v", err)
	}
	return utils.Success[assistant_api.GetAssistantKnowledgeResponse, *assistant_api.AssistantKnowledge](out)
}

func (assistantApi *assistantGrpcApi) CreateAssistantTool(ctx context.Context, atr *assistant_api.CreateAssistantToolRequest) (*assistant_api.GetAssistantToolResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantToolResponse](
			errors.New("unauthenticated request for get assistant"),
			"Please provider valid service credentials to perform CreateAssistantTool, read docs @ docs.rapida.ai",
		)
	}

	aT, err := assistantApi.
		assistantToolService.
		Create(
			ctx,
			iAuth,
			atr.GetAssistantId(),
			atr.GetName(),
			atr.GetDescription(),
			atr.GetFields().AsMap(),
			atr.GetExecutionMethod(),
			atr.GetExecutionOptions(),
		)

	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantToolResponse](err.Error())
	}

	out := &assistant_api.AssistantTool{}
	err = utils.Cast(aT, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}
	return utils.Success[assistant_api.GetAssistantToolResponse, *assistant_api.AssistantTool](out)
}

func (assistantApi *assistantGrpcApi) createAssistantKnowledge(
	ctx context.Context,
	iAuth types.SimplePrinciple,
	assistantId uint64, cepm *assistant_api.CreateAssistantKnowledgeRequest) (*internal_assistant_gorm.AssistantKnowledge, error) {
	return assistantApi.assistantKnowledgeService.Create(
		ctx,
		iAuth,
		assistantId,
		cepm.GetKnowledgeId(),
		gorm_types.RetrievalMethod(cepm.GetRetrievalMethod()),
		cepm.GetRerankerEnable(),
		cepm.GetScoreThreshold(),
		cepm.GetTopK(),
		&cepm.RerankerModelProviderId,
		&cepm.RerankerModelProviderName,
		cepm.GetAssistantKnowledgeRerankerOptions(),
	)

}

func (assistantApi *assistantGrpcApi) createAssistantTool(ctx context.Context,
	iAuth types.SimplePrinciple,
	assistantId uint64,
	atr *assistant_api.CreateAssistantToolRequest) (*internal_assistant_gorm.AssistantTool, error) {
	return assistantApi.
		assistantToolService.
		Create(
			ctx,
			iAuth,
			assistantId,
			atr.GetName(),
			atr.GetDescription(),
			atr.GetFields().AsMap(),
			atr.GetExecutionMethod(),
			atr.GetExecutionOptions(),
		)

}
