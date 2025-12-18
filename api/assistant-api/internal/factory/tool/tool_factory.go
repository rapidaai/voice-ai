package internal_tool_factory

import (
	"errors"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	internal_agent_tools "github.com/rapidaai/api/assistant-api/internal/agent/tool/local"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/commons"
)

func GetToolAction(
	logger commons.Logger,
	toolOpts *internal_assistant_entity.AssistantTool,
	communcation internal_adapter_requests.Communication,
) (internal_agent_tools.ToolCaller, error) {
	switch toolOpts.ExecutionMethod {
	case "knowledge_retrieval":
		return internal_agent_tools.NewKnowledgeRetrievalToolCaller(logger, toolOpts, communcation)
	case "api_request":
		return internal_agent_tools.NewApiRequestToolCaller(logger, toolOpts, communcation)
	case "endpoint":
		return internal_agent_tools.NewEndpointToolCaller(logger, toolOpts, communcation)
	case "put_on_hold":
		return internal_agent_tools.NewPutOnHoldToolCaller(logger, toolOpts, communcation)
	case "end_of_conversation":
		return internal_agent_tools.NewEndOfConversationCaller(logger, toolOpts, communcation)
	default:
		return nil, errors.New("illegal tool action provided")
	}
}
