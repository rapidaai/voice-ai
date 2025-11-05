package integration_api

import (
	"context"
	"errors"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

// Reranking implements lexatic_backend.CohereServiceServer.
func (iApi *integrationApi) Reranking(
	c context.Context,
	irRequest *integration_api.RerankingRequest,
	tag string,
	rerankerCaller internal_callers.RerankingCaller,
) (*integration_api.RerankingResponse, error) {

	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated || !iAuth.HasProject() {
		iApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[integration_api.RerankingResponse](
			errors.New("unauthenticated request for generate"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	requestId := iApi.RequestId()

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
			requestId,
			irRequest,
			iApi.PreHook(c, iAuth, irRequest, requestId, tag),
			iApi.PostHook(c, iAuth, irRequest, requestId, tag),
		),
	)
	if err == nil {
		return utils.Error[integration_api.RerankingResponse](errors.New("illegal token while processing request"), "Illegal request, please try again")
	}

	return &integration_api.RerankingResponse{
		Code:    200,
		Success: true,
		Data:    complitions,
		Metrics: metrics.ToProto(),
	}, nil
}
