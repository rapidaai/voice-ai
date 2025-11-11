package assistant_api

import (
	"context"
	"errors"
	"sync"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_assistant_telemetry_exporters "github.com/rapidaai/api/assistant-api/internal/telemetry/assistant/exporters"
	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/structpb"
)

// GetAllAssistantMessage implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantMessage(ctx context.Context, cepm *assistant_api.GetAllAssistantMessageRequest) (*assistant_api.GetAllAssistantMessageResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAllAssistantMessageResponse]()
	}
	cnt, epms, err := assistantApi.conversactionService.GetAllAssistantMessage(ctx,
		iAuth,
		cepm.GetAssistantId(),
		cepm.GetCriterias(),
		cepm.GetPaginate(), cepm.GetOrder(),
		internal_services.NewGetMessageOption().WithFieldSelector(cepm.GetSelectors()))
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllAssistantMessageResponse]("Unable to get the assistant for given assistant id.")
	}
	out := []*assistant_api.AssistantConversationMessage{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantMessageResponse, []*assistant_api.AssistantConversationMessage](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}

func (assistantApi *assistantGrpcApi) GetAllMessage(ctx context.Context, cepm *assistant_api.GetAllMessageRequest) (*assistant_api.GetAllMessageResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAllMessageResponse]()
	}
	cnt, epms, err := assistantApi.conversactionService.GetAllMessage(ctx,
		iAuth,
		cepm.GetCriterias(),
		cepm.GetPaginate(), cepm.GetOrder(),
		internal_services.NewGetMessageOption().WithFieldSelector(cepm.GetSelectors()))
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllMessageResponse]("Unable to get the assistant for given assistant id.")
	}
	out := []*assistant_api.AssistantConversationMessage{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllMessageResponse, []*assistant_api.AssistantConversationMessage](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}

// GetAllAssistant implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantTool(ctx context.Context, cepm *assistant_api.GetAllAssistantToolRequest) (*assistant_api.GetAllAssistantToolResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAllassistant")
		return utils.Error[assistant_api.GetAllAssistantToolResponse](
			errors.New("unauthenticated request for get all assistant skills"),
			"Please provider valid service credentials to get all assistant skills, read docs @ docs.rapida.ai",
		)
	}
	cnt, assistants, err := assistantApi.assistantToolService.GetAll(ctx, iAuth,
		cepm.GetAssistantId(),
		cepm.GetCriterias(),
		cepm.GetPaginate(),
	)
	if err != nil {
		return utils.Error[assistant_api.GetAllAssistantToolResponse](
			err,
			"Unable to get all the skill request.",
		)
	}
	out := []*assistant_api.AssistantTool{}
	err = utils.Cast(assistants, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant skill %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantToolResponse, []*assistant_api.AssistantTool](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}

func (assistantApi *assistantGrpcApi) GetAllAssistantKnowledge(ctx context.Context, cepm *assistant_api.GetAllAssistantKnowledgeRequest) (*assistant_api.GetAllAssistantKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAllassistant")
		return utils.Error[assistant_api.GetAllAssistantKnowledgeResponse](
			errors.New("unauthenticated request for get all assistant knowledge"),
			"Please provider valid service credentials to get all assistant knowledge, read docs @ docs.rapida.ai",
		)
	}
	cnt, assistants, err := assistantApi.assistantKnowledgeService.GetAll(ctx, iAuth,
		cepm.GetAssistantId(),
		cepm.GetCriterias(),
		cepm.GetPaginate(),
	)
	if err != nil {
		return utils.Error[assistant_api.GetAllAssistantKnowledgeResponse](
			err,
			"Unable to get all the skill request.",
		)
	}
	out := []*assistant_api.AssistantKnowledge{}
	err = utils.Cast(assistants, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant skill %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantKnowledgeResponse, []*assistant_api.AssistantKnowledge](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}

// GetAllAssistantConversation implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantConversation(ctx context.Context, cepm *assistant_api.GetAllAssistantConversationRequest) (*assistant_api.GetAllAssistantConversationResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAllassistant")
		return utils.Error[assistant_api.GetAllAssistantConversationResponse](
			errors.New("unauthenticated request for get all assistant skills"),
			"Please provider valid service credentials to get all assistant skills, read docs @ docs.rapida.ai",
		)
	}
	cnt, conversations, err := assistantApi.conversactionService.GetAll(ctx, iAuth,
		cepm.GetAssistantId(),
		cepm.GetCriterias(),
		cepm.GetPaginate(), internal_services.NewDefaultGetConversationOption())
	if err != nil {
		return utils.Error[assistant_api.GetAllAssistantConversationResponse](
			err,
			"Unable to get all the assistant conversation request.",
		)
	}

	out := []*assistant_api.AssistantConversation{}
	err = utils.Cast(conversations, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant conversation %v", err)
	}
	return utils.PaginatedSuccess[assistant_api.GetAllAssistantConversationResponse, []*assistant_api.AssistantConversation](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}

// GetAllConversationMessage implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllConversationMessage(ctx context.Context, cepm *assistant_api.GetAllConversationMessageRequest) (*assistant_api.GetAllConversationMessageResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAllassistant")
		return utils.Error[assistant_api.GetAllConversationMessageResponse](
			errors.New("unauthenticated request for get all assistant skills"),
			"Please provider valid service credentials to get all assistant skills, read docs @ docs.rapida.ai",
		)
	}
	cnt, messages, err := assistantApi.conversactionService.GetAllConversationMessage(ctx, iAuth,
		cepm.GetAssistantConversationId(),
		cepm.GetCriterias(),
		cepm.GetPaginate(),
		cepm.GetOrder(),
		internal_services.NewDefaultGetMessageOption())
	if err != nil {
		return utils.Error[assistant_api.GetAllConversationMessageResponse](
			err,
			"Unable to get all the conversation messages.",
		)
	}
	out := []*assistant_api.AssistantConversationMessage{}
	err = utils.Cast(messages, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant skill %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllConversationMessageResponse, []*assistant_api.AssistantConversationMessage](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}

// GetAllAssistant implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistant(ctx context.Context, cepm *assistant_api.GetAllAssistantRequest) (*assistant_api.GetAllAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAllassistant")
		return utils.Error[assistant_api.GetAllAssistantResponse](
			errors.New("unauthenticated request for get allassistant"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	cnt, assistants, err := assistantApi.assistantService.GetAll(ctx, iAuth,
		cepm.GetCriterias(),
		cepm.GetPaginate(), internal_services.NewDefaultGetAssistantOption())
	if err != nil {
		return utils.Error[assistant_api.GetAllAssistantResponse](
			err,
			"Unable to get all the assistant.",
		)
	}
	out := []*assistant_api.Assistant{}
	err = utils.Cast(assistants, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantResponse, []*assistant_api.Assistant](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}

// GetAllAssistantProviderModel implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantProvider(ctx context.Context, gaep *assistant_api.GetAllAssistantProviderRequest) (*assistant_api.GetAllAssistantProviderResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAllAssistantProviderResponse](
			errors.New("unauthenticated request for GetAllAssistantProvider"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	combinedProviders := make([]*assistant_api.GetAllAssistantProviderResponse_AssistantProvider, 0)
	var wg sync.WaitGroup
	wg.Add(1)
	// provider models
	utils.Go(ctx, func() {
		defer wg.Done()
		_, pModels, err := assistantApi.
			assistantService.
			GetAllAssistantProviderModel(ctx,
				iAuth,
				gaep.GetAssistantId(),
				gaep.GetCriterias(),
				gaep.GetPaginate())
		if err != nil {
			return
		}
		out := []*assistant_api.AssistantProviderModel{}
		err = utils.Cast(pModels, &out)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
		}
		for _, v := range out {
			combinedProviders = append(combinedProviders, &assistant_api.GetAllAssistantProviderResponse_AssistantProvider{
				AssistantProvider: &assistant_api.GetAllAssistantProviderResponse_AssistantProvider_AssistantProviderModel{
					AssistantProviderModel: v,
				},
			})
		}

	})

	wg.Add(1)
	// agentKit
	utils.Go(ctx, func() {
		defer wg.Done()
		_, pModels, err := assistantApi.
			assistantService.
			GetAllAssistantProviderAgentkit(ctx,
				iAuth,
				gaep.GetAssistantId(),
				gaep.GetCriterias(),
				gaep.GetPaginate())
		if err != nil {
			return
		}
		out := []*assistant_api.AssistantProviderAgentkit{}
		err = utils.Cast(pModels, &out)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast assistant provider agentkit %v", err)
		}
		for _, v := range out {
			combinedProviders = append(combinedProviders, &assistant_api.GetAllAssistantProviderResponse_AssistantProvider{
				AssistantProvider: &assistant_api.GetAllAssistantProviderResponse_AssistantProvider_AssistantProviderAgentkit{
					AssistantProviderAgentkit: v,
				},
			})
		}

	})

	wg.Add(1)
	utils.Go(ctx, func() {
		defer wg.Done()
		_, pModels, err := assistantApi.
			assistantService.
			GetAllAssistantProviderWebsocket(ctx,
				iAuth,
				gaep.GetAssistantId(),
				gaep.GetCriterias(),
				gaep.GetPaginate())
		if err != nil {
		}
		out := []*assistant_api.AssistantProviderWebsocket{}
		err = utils.Cast(pModels, &out)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast assistant provider websocket %v", err)
		}
		for _, v := range out {
			combinedProviders = append(combinedProviders, &assistant_api.GetAllAssistantProviderResponse_AssistantProvider{
				AssistantProvider: &assistant_api.GetAllAssistantProviderResponse_AssistantProvider_AssistantProviderWebsocket{
					AssistantProviderWebsocket: v,
				},
			})
		}

	})

	wg.Wait()
	return &assistant_api.GetAllAssistantProviderResponse{
		Code:    200,
		Success: true,
		Paginated: &assistant_api.Paginated{
			CurrentPage: gaep.GetPaginate().GetPage(),
			TotalItem:   uint32(len(combinedProviders)),
		},
		Data: combinedProviders,
	}, nil

}

func (assistantApi *assistantGrpcApi) GetAssistant(ctx context.Context, cepm *assistant_api.GetAssistantRequest) (*assistant_api.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantResponse](
			errors.New("unauthenticated request for get assistant"),
			"Please provider valid service credentials to perform GetAssistant, read docs @ docs.rapida.ai",
		)
	}

	ep, err := assistantApi.assistantService.Get(
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
	err = utils.Cast(ep, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant %v", err)
	}

	if ep.AssistantWebPluginDeployment != nil {
		out.WebPluginDeployment.Icon = &assistant_api.AssistantWebpluginDeployment_Url{
			Url: ep.AssistantWebPluginDeployment.Icon,
		}
	}

	return &assistant_api.GetAssistantResponse{
		Data:    out,
		Success: true,
		Code:    200,
	}, nil
}

func (assistantApi *assistantGrpcApi) GetAssistantConversation(ctx context.Context, cepm *assistant_api.GetAssistantConversationRequest) (*assistant_api.GetAssistantConversationResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantConversationResponse](
			errors.New("unauthenticated request for get assistant converstaion"),
			"Please provider valid service credentials to perform GetAssistantConversation, read docs @ docs.rapida.ai",
		)
	}
	ep, err := assistantApi.conversactionService.Get(ctx,
		iAuth, cepm.
			GetAssistantId(),
		cepm.
			GetId(),
		internal_services.
			NewDefaultGetConversationOption().
			WithFieldSelector(
				cepm.
					GetSelectors(),
			))
	if err != nil {
		return utils.Error[assistant_api.
			GetAssistantConversationResponse](
			err,
			"Unable to get the assistant for given assistant id.",
		)
	}
	out := &assistant_api.AssistantConversation{}
	err = utils.Cast(ep, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant %v", err)
	}
	return &assistant_api.GetAssistantConversationResponse{
		Data:    out,
		Success: true,
		Code:    200,
	}, nil
}

// GetAllAssistantWebhook implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantWebhook(ctx context.Context, cawr *assistant_api.GetAllAssistantWebhookRequest) (*assistant_api.GetAllAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAllAssistantWebhookResponse]()
	}
	cnt, epms, err := assistantApi.assistantWebhookService.GetAll(ctx,
		iAuth,
		cawr.GetAssistantId(),
		cawr.GetCriterias(),
		cawr.GetPaginate())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllAssistantWebhookResponse]("Unable to get the assistant webhooks.")
	}
	out := []*assistant_api.AssistantWebhook{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantWebhookResponse, []*assistant_api.AssistantWebhook](
		uint32(cnt),
		cawr.GetPaginate().GetPage(),
		out)
}

func (assistantApi *assistantGrpcApi) GetAssistantWebhookLog(ctx context.Context, cepm *assistant_api.GetAssistantWebhookLogRequest) (*assistant_api.GetAssistantWebhookLogResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAssistantWebhookLogRequest")
		return utils.Error[assistant_api.GetAssistantWebhookLogResponse](
			errors.New("unauthenticated request for get assistant converstaion"),
			"Please provider valid service credentials to perform GetAssistantWebhookLogRequest, read docs @ docs.rapida.ai",
		)
	}
	lg, err := assistantApi.assistantWebhookService.GetLog(
		ctx,
		iAuth,
		cepm.GetProjectId(), cepm.GetId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantWebhookLogResponse](
			err,
			"Unable to get the assistant for given assistant id.",
		)
	}
	wl := &assistant_api.AssistantWebhookLog{}
	err = utils.Cast(wl, lg)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant webhooklog to the response object")
	}

	//

	re, rs, _ := assistantApi.assistantWebhookService.GetLogObject(ctx, *iAuth.GetCurrentOrganizationId(),
		*iAuth.GetCurrentProjectId(), cepm.GetId())
	// if err != nil {
	if re != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(re)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the request %v", err)
		}
		wl.Request = s
	}
	if rs != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(rs)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the request %v", err)
		}
		wl.Response = s
	}

	return utils.Success[assistant_api.GetAssistantWebhookLogResponse, *assistant_api.AssistantWebhookLog](wl)

}

func (assistantApi *assistantGrpcApi) GetAllAssistantWebhookLog(ctx context.Context, gaar *assistant_api.GetAllAssistantWebhookLogRequest) (*assistant_api.GetAllAssistantWebhookLogResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAllAssistantWebhookLogResponse]()
	}
	cnt, epms, err := assistantApi.assistantWebhookService.GetAllLog(ctx,
		iAuth,
		gaar.GetProjectId(),
		gaar.GetCriterias(),
		gaar.GetPaginate(),
		gaar.GetOrder())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllAssistantWebhookLogResponse]("Unable to get the assistant for given assistant id.")
	}
	out := []*assistant_api.AssistantWebhookLog{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant webhook logs %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantWebhookLogResponse, []*assistant_api.AssistantWebhookLog](
		uint32(cnt),
		gaar.GetPaginate().GetPage(),
		out)
}

// GetAssistantWebhook implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAssistantWebhook(ctx context.Context, gawr *assistant_api.GetAssistantWebhookRequest) (*assistant_api.GetAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAssistantWebhookResponse]()
	}
	tlp, err := assistantApi.assistantWebhookService.Get(ctx, iAuth, gawr.GetId(), gawr.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantWebhookResponse](
			err,
			"Unable to get the webhook for given webhook id.",
		)
	}
	out := &assistant_api.AssistantWebhook{}
	err = utils.Cast(tlp, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant analysis %v", err)
	}

	return utils.Success[assistant_api.GetAssistantWebhookResponse, *assistant_api.AssistantWebhook](out)
}

func (assistantApi *assistantGrpcApi) GetAllAssistantAnalysis(ctx context.Context, cawr *assistant_api.GetAllAssistantAnalysisRequest) (*assistant_api.GetAllAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAllAssistantAnalysisResponse]()
	}
	cnt, epms, err := assistantApi.assistantAnalysisService.GetAll(ctx,
		iAuth,
		cawr.GetAssistantId(),
		cawr.GetCriterias(),
		cawr.GetPaginate())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllAssistantAnalysisResponse]("Unable to get the assistant webhooks.")
	}
	out := []*assistant_api.AssistantAnalysis{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant analysis %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantAnalysisResponse, []*assistant_api.AssistantAnalysis](
		uint32(cnt),
		cawr.GetPaginate().GetPage(),
		out)
}

func (assistantApi *assistantGrpcApi) GetAssistantAnalysis(ctx context.Context, gawr *assistant_api.GetAssistantAnalysisRequest) (*assistant_api.GetAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAssistantAnalysisResponse]()
	}
	tlp, err := assistantApi.assistantAnalysisService.Get(ctx, iAuth, gawr.GetId(), gawr.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantAnalysisResponse](
			err,
			"Unable to get the analysis for given webhook id.",
		)
	}
	out := &assistant_api.AssistantAnalysis{}
	err = utils.Cast(tlp, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast analysis %v", err)
	}
	return utils.Success[assistant_api.GetAssistantAnalysisResponse, *assistant_api.AssistantAnalysis](out)
}

func (assistantApi *assistantGrpcApi) GetAssistantKnowledge(ctx context.Context, gawr *assistant_api.GetAssistantKnowledgeRequest) (*assistant_api.GetAssistantKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAssistantKnowledgeResponse]()
	}
	tlp, err := assistantApi.assistantKnowledgeService.Get(ctx, iAuth, gawr.GetId(), gawr.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantKnowledgeResponse](
			err,
			"Unable to get the Knowledge for given webhook id.",
		)
	}
	out := &assistant_api.AssistantKnowledge{}
	err = utils.Cast(tlp, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast Knowledge %v", err)
	}
	return utils.Success[assistant_api.GetAssistantKnowledgeResponse, *assistant_api.AssistantKnowledge](out)
}

func (assistantApi *assistantGrpcApi) GetAssistantTool(ctx context.Context, gawr *assistant_api.GetAssistantToolRequest) (*assistant_api.GetAssistantToolResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAssistantToolResponse]()
	}
	tlp, err := assistantApi.assistantToolService.Get(ctx, iAuth, gawr.GetId(), gawr.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantToolResponse](
			err,
			"Unable to get the tool for given webhook id.",
		)
	}
	out := &assistant_api.AssistantTool{}
	err = utils.Cast(tlp, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast tool %v", err)
	}
	return utils.Success[assistant_api.GetAssistantToolResponse, *assistant_api.AssistantTool](out)
}

func (assistantApi *assistantGrpcApi) GetAssistantToolLog(ctx context.Context, cepm *assistant_api.GetAssistantToolLogRequest) (*assistant_api.GetAssistantToolLogResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAssistantToolLogRequest")
		return utils.Error[assistant_api.GetAssistantToolLogResponse](
			errors.New("unauthenticated request for get assistant converstaion"),
			"Please provider valid service credentials to perform GetAssistantToolLogRequest, read docs @ docs.rapida.ai",
		)
	}
	lg, err := assistantApi.assistantToolService.GetLog(
		ctx,
		iAuth,
		cepm.GetProjectId(), cepm.GetId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantToolLogResponse](
			err,
			"Unable to get the assistant for given assistant id.",
		)
	}
	wl := &assistant_api.AssistantToolLog{}
	err = utils.Cast(lg, wl)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant ToolLog to the response object")
	}
	re, rs, _ := assistantApi.assistantToolService.GetLogObject(ctx, *iAuth.GetCurrentOrganizationId(),
		*iAuth.GetCurrentProjectId(), cepm.GetId())
	// if err != nil {
	if re != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(re)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the request %v", err)
		}
		wl.Request = s
	}
	if rs != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(rs)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the request %v", err)
		}
		wl.Response = s
	}

	return utils.Success[assistant_api.GetAssistantToolLogResponse, *assistant_api.AssistantToolLog](wl)

}

func (assistantApi *assistantGrpcApi) GetAllAssistantToolLog(ctx context.Context, gaar *assistant_api.GetAllAssistantToolLogRequest) (*assistant_api.GetAllAssistantToolLogResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAllAssistantToolLogResponse]()
	}
	cnt, epms, err := assistantApi.assistantToolService.GetAllLog(ctx,
		iAuth,
		gaar.GetProjectId(),
		gaar.GetCriterias(),
		gaar.GetPaginate(),
		gaar.GetOrder())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllAssistantToolLogResponse]("Unable to get the assistant for given assistant id.")
	}
	out := []*assistant_api.AssistantToolLog{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant webhook logs %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantToolLogResponse, []*assistant_api.AssistantToolLog](
		uint32(cnt),
		gaar.GetPaginate().GetPage(),
		out)
}

// GetAllAssistantConversationTelemetry implements lexatic_backend.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantTelemetry(ctx context.Context, request *assistant_api.GetAllAssistantTelemetryRequest) (*assistant_api.GetAllAssistantTelemetryResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAllAssistantTelemetryResponse]()
	}

	otelExporter := internal_assistant_telemetry_exporters.NewOpensearchAssistantTraceExporter(
		assistantApi.logger,
		&assistantApi.cfg.AppConfig,
		assistantApi.opensearch,
	)
	cnt, ot, err := otelExporter.Get(ctx, iAuth, request.Criterias, request.Paginate)
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllAssistantTelemetryResponse]("Unable to get the assistant telemetry.")
	}
	out := make([]*assistant_api.Telemetry, 0)
	for _, v := range ot {
		out = append(out, v.ToProto())
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantTelemetryResponse, []*assistant_api.Telemetry](
		uint32(cnt),
		request.GetPaginate().GetPage(),
		out)

}
