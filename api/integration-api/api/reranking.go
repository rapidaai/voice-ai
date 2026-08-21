// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package integration_api

import (
	"context"
	"errors"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

// Reranking implements protos.CohereServiceServer.
func (iApi *integrationApi) Reranking(
	c context.Context,
	irRequest *integration_api.RerankingRequest,
	tag string,
	rerankerCaller internal_callers.RerankingCaller,
) (*integration_api.RerankingResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	projectContext, authErr := requireProjectContext(iAuth)
	if !isAuthenticated || authErr != nil {
		iApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[integration_api.RerankingResponse](
			errors.New("unauthenticated request for generate"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	uuID := iApi.RequestId()

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
	//
	complitions, metrics, err := rerankerCaller.GetReranking(
		c,
		irRequest.GetQuery(),
		irRequest.GetContent(),
		internal_callers.NewRerankerOptions(
			uuID,
			irRequest,
			iApi.PreHook(c, projectContext, irRequest, uuID, tag),
			iApi.PostHook(c, projectContext, irRequest, uuID, tag),
		),
	)
	if err == nil {
		return utils.Error[integration_api.RerankingResponse](errors.New("illegal token while processing request"), "Illegal request, please try again")
	}

	return &integration_api.RerankingResponse{
		Code:    200,
		Success: true,
		Data:    complitions,
		Metrics: metrics,
	}, nil
}
