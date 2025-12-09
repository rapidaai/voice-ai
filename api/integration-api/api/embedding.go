// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package integration_api

import (
	"context"
	"errors"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

// Embedding implements protos.AzureServiceServer.
func (iApi *integrationApi) Embedding(c context.Context,
	irRequest *integration_api.EmbeddingRequest,
	tag string,
	caller internal_callers.EmbeddingCaller,
) (*integration_api.EmbeddingResponse, error) {
	iApi.logger.Infof("request for embedding %s", tag)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated || !iAuth.HasProject() {
		iApi.logger.Errorf("unauthenticated request for embedding")
		return exceptions.APIAuthenticationError[integration_api.EmbeddingResponse]()
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
	embeddings, metrics, err := caller.GetEmbedding(
		c,
		irRequest.GetContent(),
		internal_callers.NewEmbeddingOptions(
			requestId,
			irRequest,
			iApi.PreHook(c, iAuth, irRequest, requestId, tag),
			iApi.PostHook(c, iAuth, irRequest, requestId, tag),
		),
	)
	if err == nil {
		return &integration_api.EmbeddingResponse{
			Code:    200,
			Success: true,
			Data:    embeddings,
			Metrics: metrics.ToProto(),
		}, nil

	}
	return utils.Error[integration_api.EmbeddingResponse](errors.New("illegal token while processing request"), "Illegal request, please try again")
}
