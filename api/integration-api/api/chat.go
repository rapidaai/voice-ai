// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package integration_api

import (
	"context"
	"errors"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamChat handles streaming chat requests to a large language model.
//
// This method:
// 1. Authenticates the request using the provided context.
// 2. Initiates a streaming chat completion using the provided LLM caller.
//utions.
// 3. Processes the streaming responses, including content, metrics, and errors.
// 4. Sends formatted responses back to the client using the provided send function.
//
// Parameters:
// - irRequest: The chat request containing model and conversation details.
// - context: The context for the request, used for authentication and cancellation.
// - tag: A string identifier for the request, used for logging.
// - llmCaller: The interface to call the large language model.
// - send: A function to send responses back to the client.
//
// The method uses channels to handle concurrent processing of content, metrics,
// and errors from the LLM. It continues to stream responses until the output
// channel is closed or an error occurs.
//
// Returns an error if authentication fails or if there's an issue sending responses.

func (iApi *integrationApi) StreamChat(
	irRequest *protos.ChatRequest,
	context context.Context,
	providerName string,
	llmCaller internal_callers.LargeLanguageCaller,
	send func(*protos.ChatResponse) error) error {

	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(context)
	if !isAuthenticated || !iAuth.HasProject() {
		iApi.logger.Errorf("unauthenticated request for invoke")
		return status.Error(codes.Unauthenticated, "Please provide valid service credentials to perform invoke.")
	}
	requestId := iApi.RequestId()
	if irRequest.AdditionalData == nil {
		irRequest.AdditionalData = map[string]string{}
	}

	irRequest.AdditionalData["provider_name"] = providerName
	model, ok := irRequest.ModelParameters["model.name"]
	if ok {
		mdl, err := utils.AnyToString(model)
		if err == nil {
			irRequest.AdditionalData["model_name"] = mdl
		}
	}

	modelID, ok := irRequest.ModelParameters["model.id"]
	if ok {
		mdlID, err := utils.AnyToString(modelID)
		if err == nil {
			irRequest.AdditionalData["model_id"] = mdlID
		}
	}

	source, ok := utils.GetClientSource(context)
	if ok {
		irRequest.AdditionalData["source"] = source.Get()
	}

	clientEnv, ok := utils.GetClientEnvironment(context)
	if ok {
		irRequest.AdditionalData["env"] = clientEnv.Get()
	}

	clientRegion, ok := utils.GetClientRegion(context)
	if ok {
		irRequest.AdditionalData["region"] = clientRegion.Get()
	}

	return llmCaller.StreamChatCompletion(
		context,
		irRequest.GetConversations(),
		internal_callers.NewChatOptions(
			requestId,
			irRequest,
			iApi.PreHook(context, iAuth, irRequest, requestId, providerName),
			iApi.PostHook(context, iAuth, irRequest, requestId, providerName),
		),
		func(content types.Message) error {
			return send(&protos.ChatResponse{
				Success:   true,
				RequestId: requestId,
				Data:      content.ToProto(),
			})
		},
		func(content *types.Message, mtx types.Metrics) error {
			return send(&protos.ChatResponse{
				Success:   true,
				RequestId: requestId,
				Metrics:   mtx.ToProto(),
				Data:      content.ToProto(),
			})
		},
		func(err error) {
			send(&protos.ChatResponse{
				Success:   false,
				Code:      400,
				RequestId: requestId,
				Error: &protos.Error{
					ErrorCode:    uint64(400),
					ErrorMessage: err.Error(),
					HumanMessage: err.Error(),
				},
			})
		},
	)

}

func (iApi *integrationApi) Chat(
	c context.Context,
	irRequest *protos.ChatRequest,
	tag string,
	caller internal_callers.LargeLanguageCaller,
) (*protos.ChatResponse, error) {
	iApi.logger.Infof("Chat from grpc with provider %s", tag)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated || !iAuth.HasProject() {
		iApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[protos.ChatResponse](
			errors.New("unauthenticated request for chat"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	requestId := iApi.RequestId()
	if irRequest.AdditionalData == nil {
		irRequest.AdditionalData = map[string]string{}
	}

	irRequest.AdditionalData["provider_name"] = tag
	model, ok := irRequest.ModelParameters["model.name"]
	if ok {
		mdl, err := utils.AnyToString(model)
		if err == nil {
			irRequest.AdditionalData["model_name"] = mdl
		}
	}

	modelID, ok := irRequest.ModelParameters["model.id"]
	if ok {
		mdlID, err := utils.AnyToString(modelID)
		if err == nil {
			irRequest.AdditionalData["model_id"] = mdlID
		}
	}
	source, ok := utils.GetClientSource(c)
	if ok {
		irRequest.AdditionalData["source"] = source.Get()
	}

	clientEnv, ok := utils.GetClientEnvironment(c)
	if ok {
		irRequest.AdditionalData["env"] = clientEnv.Get()
	}

	clientRegion, ok := utils.GetClientRegion(c)
	if ok {
		irRequest.AdditionalData["region"] = clientRegion.Get()
	}

	completions, metrics, err := caller.GetChatCompletion(
		c,
		irRequest.GetConversations(),
		internal_callers.NewChatOptions(
			requestId,
			irRequest,
			iApi.PreHook(c, iAuth, irRequest, requestId, tag),
			iApi.PostHook(c, iAuth, irRequest, requestId, tag),
		),
	)
	if err != nil {
		return utils.Error[protos.ChatResponse](err, err.Error())
	}
	return &protos.ChatResponse{
		Code:    200,
		Success: true,
		Data:    completions.ToProto(),
		Metrics: metrics.ToProto(),
	}, nil

}
