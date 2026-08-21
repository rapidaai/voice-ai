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
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/structpb"
)

func (assistantApi *assistantGrpcApi) GetAssistantHTTPLog(ctx context.Context, req *protos.GetAssistantHTTPLogRequest) (*protos.GetAssistantHTTPLogResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	projectContext, authErr := types.RequireProject(iAuth)
	if !isAuthenticated || authErr != nil {
		assistantApi.logger.Errorf("unauthenticated request for GetAssistantHTTPLogRequest")
		return utils.Error[protos.GetAssistantHTTPLogResponse](
			errors.New("unauthenticated request for get assistant http log"),
			"Please provide valid service credentials to perform GetAssistantHTTPLogRequest, read docs @ docs.rapida.ai",
		)
	}

	logRecord, err := assistantApi.assistantHTTPLogService.GetLog(
		ctx,
		iAuth,
		req.GetProjectId(),
		req.GetId(),
	)
	if err != nil {
		return utils.Error[protos.GetAssistantHTTPLogResponse](
			err,
			"Unable to get the HTTP log for given id.",
		)
	}

	out := &protos.AssistantHTTPLog{}
	if err := utils.Cast(logRecord, out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant http log to response object")
	}

	requestData, responseData, _ := assistantApi.assistantHTTPLogService.GetLogObject(
		ctx,
		projectContext.OrganizationID,
		projectContext.ProjectID,
		req.GetId(),
	)
	if requestData != nil {
		s := &structpb.Struct{}
		if err := s.UnmarshalJSON(requestData); err != nil {
			assistantApi.logger.Errorf("unable to cast the request %v", err)
		}
		out.Request = s
	}
	if responseData != nil {
		s := &structpb.Struct{}
		if err := s.UnmarshalJSON(responseData); err != nil {
			assistantApi.logger.Errorf("unable to cast the response %v", err)
		}
		out.Response = s
	}

	return utils.Success[protos.GetAssistantHTTPLogResponse, *protos.AssistantHTTPLog](out)
}
