package assistant_talk_api

import (
	"context"
	"errors"
	"fmt"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
	lexatic_backend "github.com/rapidaai/protos"
)

func (cApi *ConversationApi) CreateMessageMetric(ctx context.Context, cer *assistant_api.CreateMessageMetricRequest) (*assistant_api.CreateMessageMetricResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		return utils.Error[assistant_api.CreateMessageMetricResponse](
			errors.New("unauthenticated request for CreateMessageMetric"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	mtr := make([]*types.Metric, 0)
	for _, v := range cer.GetMetrics() {
		mtr = append(mtr, &types.Metric{
			Name:        fmt.Sprintf("custom.%s", v.GetName()),
			Value:       v.GetValue(),
			Description: v.GetDescription(),
		})
	}
	val, err := cApi.assistantConversationService.ApplyMessageMetrics(
		ctx,
		iAuth,
		cer.GetAssistantConversationId(),
		cer.GetMessageId(),
		mtr,
	)
	if err != nil {
		return exceptions.InternalServerError[lexatic_backend.CreateMessageMetricResponse](
			err,
			"Unable to get all the assistant for the conversaction.",
		)
	}
	return utils.Success[lexatic_backend.CreateMessageMetricResponse](val)
}

// ConversationFeedback implements lexatic_backend.TalkServiceServer.
func (cApi *ConversationGrpcApi) CreateConversationMetric(ctx context.Context, cfr *assistant_api.CreateConversationMetricRequest) (*assistant_api.CreateConversationMetricResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		return utils.Error[assistant_api.CreateConversationMetricResponse](
			errors.New("unauthenticated request for CreateConversationMetric"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	val, err := cApi.assistantConversationService.CreateCustomConversationMetric(
		ctx,
		iAuth,
		cfr.GetAssistantId(),
		cfr.GetAssistantConversationId(),
		cfr.GetMetrics(),
	)
	if err != nil {
		return exceptions.InternalServerError[lexatic_backend.CreateConversationMetricResponse](
			err,
			"Unable to get all the assistant for the conversaction.",
		)
	}
	out := &lexatic_backend.AssistantConversation{}
	err = utils.Cast(val, out)
	if err != nil {
		cApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}
	return utils.Success[lexatic_backend.CreateConversationMetricResponse](out)
}
