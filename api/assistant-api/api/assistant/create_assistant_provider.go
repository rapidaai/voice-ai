// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"
	"fmt"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/encoding/protojson"
)

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
