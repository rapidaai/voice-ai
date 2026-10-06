// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package node

import (
	"context"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

type MessageHandler struct{}

func (handler MessageHandler) Type() string {
	return schema.NodeTypeMessage
}

func (handler MessageHandler) Close(context.Context) error {
	return nil
}

func (handler MessageHandler) Execute(ctx context.Context, request Request) (Result, error) {
	message := request.Node.StringConfig("message")
	if message != "" {
		_ = request.Communication.OnPacket(ctx,
			internal_type.LLMResponseDeltaPacket{ContextID: request.ContextID, Text: message},
		)
		request.RuntimeState.RecordNodeOutputValue(request.Node.ID, "response", message)
	}

	postDelayMilliseconds := request.Node.IntConfig("post_delay_ms", 0)
	if postDelayMilliseconds <= 0 {
		return Result{RouteHandles: []string{"response", "next"}, ResponseText: message}, nil
	}

	timer := time.NewTimer(time.Duration(postDelayMilliseconds) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-timer.C:
		return Result{RouteHandles: []string{"response", "next"}, ResponseText: message}, nil
	}
}
