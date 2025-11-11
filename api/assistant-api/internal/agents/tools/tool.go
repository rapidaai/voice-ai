package internal_agent_tools

import (
	"context"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type ToolCaller interface {
	// tool call id
	Id() uint64

	//
	Name() string

	//
	Definition() (*lexatic_backend.FunctionDefinition, error)

	//
	ExecutionMethod() string

	//
	Call(
		ctx context.Context,
		messageId string,
		args string,
		communication internal_adapter_requests.Communication,
	) (map[string]interface{}, []*types.Metric)
}

type toolCaller struct {
	logger      commons.Logger
	toolOptions *internal_assistant_gorm.AssistantTool
}

func (executor *toolCaller) Name() string {
	return executor.toolOptions.Name
}

func (executor *toolCaller) Id() uint64 {
	return executor.toolOptions.Id
}

func (executor *toolCaller) ExecutionMethod() string {
	return executor.toolOptions.ExecutionMethod
}

func (executor *toolCaller) Definition() (*lexatic_backend.FunctionDefinition, error) {
	definition := &lexatic_backend.FunctionDefinition{
		Name:        executor.toolOptions.Name,
		Description: executor.toolOptions.Description,
		Parameters:  &lexatic_backend.FunctionParameter{},
	}
	if err := utils.Cast(executor.toolOptions.Fields, definition.Parameters); err != nil {
		return nil, err
	}
	return definition, nil
}

func (executor *toolCaller) Result(msg string, success bool) map[string]interface{} {
	if success {
		return map[string]interface{}{
			"data":    msg,
			"success": true,
			"status":  "SUCCESS",
		}
	} else {
		return map[string]interface{}{
			"error":   msg,
			"success": false,
			"status":  "FAIL",
		}
	}
}
