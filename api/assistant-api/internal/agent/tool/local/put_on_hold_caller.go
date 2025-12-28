// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_local_tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type putOnHoldToolCaller struct {
	toolCaller
	maxHoldTime uint64
}

func (tc *putOnHoldToolCaller) argument(args string) uint64 {
	var input map[string]interface{}
	err := json.Unmarshal([]byte(args), &input)
	if err != nil {
		tc.logger.Debugf("illegal input from llm check and pushing the llm response as incomplete %v", args)
		return tc.maxHoldTime
	}
	var duration uint64
	switch v := input["duration"].(type) {
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return tc.maxHoldTime
		}
		return utils.MaxUint64(duration, parsed)
	case float64:
		return utils.MaxUint64(duration, uint64(v))
	case int:
		return utils.MaxUint64(duration, uint64(v))
	case uint64:
		return utils.MaxUint64(duration, v)
	}
	return tc.maxHoldTime

}

func (afkTool *putOnHoldToolCaller) Call(
	ctx context.Context,
	messageId string,
	args string,
	communication internal_adapter_requests.Communication,
) (map[string]interface{}, []*types.Metric) {
	start := time.Now()
	metrics := make([]*types.Metric, 0)
	// duration := afkTool.argument(args)
	err := communication.Notify(
		ctx,
		&protos.AssistantMessagingResponse_Action{
			Action: &protos.AssistantConversationAction{
				Name:   afkTool.Name(),
				Action: protos.AssistantConversationAction_PUT_ON_HOLD,
			},
		},
	)
	metrics = append(metrics, types.NewTimeTakenMetric(time.Since(start)))
	if err != nil {
		return afkTool.Result("Unable to disconnect. Please try again later.", false), metrics
	}
	return afkTool.Result("Disconnected successfully.", true), metrics
}

func NewPutOnHoldToolCaller(
	logger commons.Logger,
	toolOptions *internal_assistant_entity.AssistantTool,
	communication internal_adapter_requests.Communication,
) (ToolCaller, error) {

	opts := toolOptions.GetOptions()
	var maxHoldTime uint64
	switch v := opts["tool.max_hold_time"].(type) {
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("tool.endpoint_id is not a valid number: %v", err)
		}
		maxHoldTime = parsed
	case float64:
		maxHoldTime = uint64(v)
	case int:
		maxHoldTime = uint64(v)
	case uint64:
		maxHoldTime = v
	default:
		return nil, fmt.Errorf("tool.max_hold_time is not a recognized type, got %T", v)
	}

	return &putOnHoldToolCaller{
		toolCaller: toolCaller{
			logger:      logger,
			toolOptions: toolOptions,
		},
		maxHoldTime: maxHoldTime,
	}, nil
}
