// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_local

import (
	"context"
	"encoding/json"

	internal_tool "github.com/rapidaai/api/assistant-api/internal/agent/executor/tool/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type stageTransitionCaller struct {
	logger        commons.Logger
	communication internal_type.Communication
}

func (caller *stageTransitionCaller) Name() string {
	return "transition_stage"
}

func (caller *stageTransitionCaller) Id() uint64 {
	return 0
}

func (caller *stageTransitionCaller) ExecutionMethod() string {
	return "stage_transition"
}

func (caller *stageTransitionCaller) Definition() (*protos.FunctionDefinition, error) {
	return &protos.FunctionDefinition{
		Name:        "transition_stage",
		Description: "Transition to a different conversation stage. Use stage name (e.g., 'order_placement', 'identity_check'). Use after completing a task to switch prompts.",
		Parameters: &protos.FunctionParameter{
			Type:     "object",
			Required: []string{"stage_name"},
			Properties: map[string]*protos.FunctionParameterProperty{
				"stage_name": {
					Type:        "string",
					Description: "The name of the stage to transition to (e.g., 'order_placement', 'identity_check', 'verification')",
				},
			},
		},
	}, nil
}

func (caller *stageTransitionCaller) Call(ctx context.Context, contextID, toolId string, args map[string]interface{}, communication internal_type.Communication) internal_tool.ToolCallResult {
	stageName, ok := args["stage_name"].(string)
	if !ok || stageName == "" {
		return internal_tool.Result("Error: stage_name is required", false)
	}

	// Emit a directive packet with stage transition info
	// Use END_CONVERSATION directive as a carrier, the actual stage_name will be in arguments
	// The conversation handler needs to check arguments for TRANSITION_STAGE
	communication.OnPacket(ctx, internal_type.DirectivePacket{
		Directive: protos.ConversationDirective_END_CONVERSATION,
		Arguments: map[string]interface{}{
			"transition_stage": stageName,
		},
		ContextID: contextID,
	})

	result := map[string]interface{}{
		"success":    true,
		"stage_name": stageName,
		"message":    "Stage transition requested: " + stageName,
	}
	resultJSON, _ := json.Marshal(result)
	return internal_tool.Result(string(resultJSON), true)
}

func NewStageTransitionCaller(ctx context.Context, logger commons.Logger, communication internal_type.Communication) internal_tool.ToolCaller {
	return &stageTransitionCaller{
		logger:        logger,
		communication: communication,
	}
}
