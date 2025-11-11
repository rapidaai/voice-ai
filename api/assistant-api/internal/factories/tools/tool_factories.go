package internal_tool_factories

import (
	"errors"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	internal_agent_tools "github.com/rapidaai/api/assistant-api/internal/agents/tools"
	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	"github.com/rapidaai/pkg/commons"
)

func GetToolAction(
	logger commons.Logger,
	toolOpts *internal_assistant_gorm.AssistantTool,
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
