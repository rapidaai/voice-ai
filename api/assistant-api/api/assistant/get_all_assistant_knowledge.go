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
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

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
