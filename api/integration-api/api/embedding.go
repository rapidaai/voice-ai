// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package integration_api

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	integration_api "github.com/rapidaai/protos"
)

// Embedding implements protos.AzureServiceServer.
func (iApi *integrationApi) Embedding(c context.Context,
	irRequest *integration_api.EmbeddingRequest,
	tag string,
	caller internal_callers.EmbeddingCaller,
) (*integration_api.EmbeddingResponse, error) {
	iApi.logger.Infof("request for embedding %s", tag)
	iAuth, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
	}
	if !validator.OneOf(iAuth.Subject().Type, authentication.IdentityUser, authentication.IdentityProjectCredential, authentication.IdentityService) {
		return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
	}
	projectContext := iAuth.Tenant()
	if projectContext.OrganizationID == 0 || projectContext.ProjectID == 0 {
		return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
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
			iApi.PreHook(c, projectContext, irRequest, requestId, tag),
			iApi.PostHook(c, projectContext, irRequest, requestId, tag),
		),
	)
	if err == nil {
		return &integration_api.EmbeddingResponse{
			Code:    200,
			Success: true,
			Data:    embeddings,
			Metrics: metrics,
		}, nil
	}
	return utils.Error[integration_api.EmbeddingResponse](errors.New("illegal token while processing request"), "Illegal request, please try again")
}
