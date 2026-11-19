// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package knowledge_api

import (
	"context"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
)

// UpdateKnowledgeDetail implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) UpdateKnowledgeDetail(ctx context.Context, cer *knowledge_api.UpdateKnowledgeDetailRequest) (*knowledge_api.GetKnowledgeResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	kn, err := knowledgeApi.knowledgeService.UpdateKnowledgeDetail(ctx, iAuth, cer.GetKnowledgeId(), cer.GetName(), cer.GetDescription())
	if err != nil {
		knowledgeApi.logger.Errorf("unable to update knowledge details with error %v", err)
		return utils.Error[knowledge_api.GetKnowledgeResponse](
			err,
			"Unable to update knowledge details, please try again.",
		)
	}

	_kn, err := knowledgeApi.knowledgeService.Get(ctx, iAuth, kn.Id)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to get knowledge with error %v", err)
		return utils.Error[knowledge_api.GetKnowledgeResponse](
			err,
			"Unable to get knowledge, please try again later.",
		)
	}
	out := &knowledge_api.Knowledge{}
	err = utils.Cast(_kn, out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast the knowledge model to the response object")
	}
	return utils.Success[knowledge_api.GetKnowledgeResponse, *knowledge_api.Knowledge](out)
}
