// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/structpb"
)

func (assistantApi *assistantGrpcApi) GetAssistantHTTPLog(ctx context.Context, req *protos.GetAssistantHTTPLogRequest) (*protos.GetAssistantHTTPLogResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
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
		iAuth,
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
