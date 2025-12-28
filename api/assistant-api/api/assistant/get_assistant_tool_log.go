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

func (assistantApi *assistantGrpcApi) GetAssistantToolLog(ctx context.Context, cepm *protos.GetAssistantToolLogRequest) (*protos.GetAssistantToolLogResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAssistantToolLogRequest")
		return utils.Error[protos.GetAssistantToolLogResponse](
			errors.New("unauthenticated request for get assistant converstaion"),
			"Please provider valid service credentials to perform GetAssistantToolLogRequest, read docs @ docs.rapida.ai",
		)
	}
	lg, err := assistantApi.assistantToolService.GetLog(
		ctx,
		iAuth,
		cepm.GetProjectId(), cepm.GetId())
	if err != nil {
		return utils.Error[protos.GetAssistantToolLogResponse](
			err,
			"Unable to get the assistant for given assistant id.",
		)
	}
	wl := &protos.AssistantToolLog{}
	err = utils.Cast(lg, wl)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant ToolLog to the response object")
	}
	re, rs, _ := assistantApi.assistantToolService.GetLogObject(ctx, *iAuth.GetCurrentOrganizationId(),
		*iAuth.GetCurrentProjectId(), cepm.GetId())
	// if err != nil {
	if re != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(re)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the request %v", err)
		}
		wl.Request = s
	}
	if rs != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(rs)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the request %v", err)
		}
		wl.Response = s
	}

	return utils.Success[protos.GetAssistantToolLogResponse, *protos.AssistantToolLog](wl)

}
