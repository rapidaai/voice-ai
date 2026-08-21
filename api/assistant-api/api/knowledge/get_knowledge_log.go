// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package knowledge_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/structpb"
)

func (knowledgeApi *knowledgeGrpcApi) GetKnowledgeLog(ctx context.Context, cepm *knowledge_api.GetKnowledgeLogRequest) (*knowledge_api.GetKnowledgeLogResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	projectContext, authErr := types.RequireProject(iAuth)
	if !isAuthenticated || authErr != nil {
		knowledgeApi.logger.Errorf("unauthenticated request for GetKnowledgeLogRequest")
		return utils.Error[knowledge_api.GetKnowledgeLogResponse](
			errors.New("unauthenticated request for get assistant converstaion"),
			"Please provider valid service credentials to perform GetKnowledgeLogRequest, read docs @ docs.rapida.ai",
		)
	}
	lg, err := knowledgeApi.knowledgeService.GetLog(
		ctx,
		iAuth,
		cepm.GetProjectId(), cepm.GetId())
	if err != nil {
		return utils.Error[knowledge_api.GetKnowledgeLogResponse](
			err,
			"Unable to get the assistant for given assistant id.",
		)
	}
	wl := &knowledge_api.KnowledgeLog{}
	err = utils.Cast(lg, wl)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast the assistant ToolLog to the response object")
	}

	//

	re, rs, _ := knowledgeApi.knowledgeService.GetLogObject(ctx, projectContext.OrganizationID,
		projectContext.ProjectID, cepm.GetId())
	// if err != nil {
	if re != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(re)
		if err != nil {
			knowledgeApi.logger.Errorf("unable to cast the request %v", err)
		}
		wl.Request = s
	}
	if rs != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(rs)
		if err != nil {
			knowledgeApi.logger.Errorf("unable to cast the request %v", err)
		}
		wl.Response = s
	}

	return utils.Success[knowledge_api.GetKnowledgeLogResponse, *knowledge_api.KnowledgeLog](wl)

}
