// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) GetAssistantKnowledge(ctx context.Context, gawr *protos.GetAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	tlp, err := assistantApi.assistantKnowledgeService.Get(ctx, iAuth, gawr.GetId(), gawr.GetAssistantId())
	if err != nil {
		return utils.Error[protos.GetAssistantKnowledgeResponse](
			err,
			"Unable to get the Knowledge for given webhook id.",
		)
	}
	out := &protos.AssistantKnowledge{}
	err = utils.Cast(tlp, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast Knowledge %v", err)
	}
	return utils.Success[protos.GetAssistantKnowledgeResponse, *protos.AssistantKnowledge](out)
}
