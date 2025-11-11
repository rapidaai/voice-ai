package internal_agent_tools

import (
	"context"
	"time"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	lexatic_backend "github.com/rapidaai/protos"
)

type endOfConversationCaller struct {
	toolCaller
}

func (afkTool *endOfConversationCaller) Call(
	ctx context.Context,
	messageId string,
	args string,
	communication internal_adapter_requests.Communication,
) (map[string]interface{}, []*types.Metric) {
	start := time.Now()
	metrics := make([]*types.Metric, 0)
	err := communication.
		Notify(
			ctx,
			&lexatic_backend.AssistantMessagingResponse_DisconnectAction{},
		)
	metrics = append(metrics, types.NewTimeTakenMetric(time.Since(start)))
	if err != nil {
		return afkTool.Result("Unable to disconnect. Please try again later.", false), metrics
	}
	return afkTool.Result("Disconnected successfully.", true), metrics
}

func NewEndOfConversationCaller(
	logger commons.Logger,
	toolOptions *internal_assistant_gorm.AssistantTool,
	communcation internal_adapter_requests.Communication,
) (ToolCaller, error) {
	return &endOfConversationCaller{
		toolCaller: toolCaller{
			logger:      logger,
			toolOptions: toolOptions,
		},
	}, nil
}
