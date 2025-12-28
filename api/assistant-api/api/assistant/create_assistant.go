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
