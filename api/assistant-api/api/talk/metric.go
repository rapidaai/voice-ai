// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
	protos "github.com/rapidaai/protos"
)

func (cApi *ConversationApi) CreateMessageMetric(ctx context.Context, cer *assistant_api.CreateMessageMetricRequest) (*assistant_api.CreateMessageMetricResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	val, err := cApi.assistantConversationService.CreateOrUpdateMessageMetrics(
		ctx,
		iAuth,
		cer.GetAssistantConversationId(),
		cer.GetMessageId(),
		cer.GetMetrics(),
	)
	if err != nil {
		return exceptions.InternalServerError[protos.CreateMessageMetricResponse](
			err,
			"Unable to get all the assistant for the conversaction.",
		)
	}
	return utils.Success[protos.CreateMessageMetricResponse](val)
}

// ConversationFeedback implements protos.TalkServiceServer.
func (cApi *ConversationGrpcApi) CreateConversationMetric(ctx context.Context, cfr *assistant_api.CreateConversationMetricRequest) (*assistant_api.CreateConversationMetricResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	val, err := cApi.assistantConversationService.CreateCustomConversationMetric(
		ctx,
		iAuth,
		cfr.GetAssistantId(),
		cfr.GetAssistantConversationId(),
		cfr.GetMetrics(),
	)
	if err != nil {
		return exceptions.InternalServerError[protos.CreateConversationMetricResponse](
			err,
			"Unable to get all the assistant for the conversaction.",
		)
	}
	out := &protos.AssistantConversation{}
	err = utils.Cast(val, out)
	if err != nil {
		cApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}
	return utils.Success[protos.CreateConversationMetricResponse](out)
}
