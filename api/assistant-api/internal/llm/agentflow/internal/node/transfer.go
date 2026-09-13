// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package node

import (
	"context"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
)

type TransferHandler struct{}

func (handler TransferHandler) Type() string {
	return schema.NodeTypeTransfer
}

func (handler TransferHandler) Close(context.Context) error {
	return nil
}

func (handler TransferHandler) Execute(ctx context.Context, request Request) (Result, error) {
	transferMessage := request.Node.StringConfig("transfer_message")
	if transferMessage != "" {
		_ = request.Communication.OnPacket(ctx,
			internal_type.LLMResponseDeltaPacket{ContextID: request.ContextID, Text: transferMessage},
		)
	}

	err := request.Communication.OnPacket(ctx, internal_type.LLMToolCallPacket{
		ContextID: request.ContextID,
		Name:      "transfer_call",
		Action:    protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Arguments: map[string]string{
			"transfer_to":          request.Node.StringConfig("transfer_to"),
			"post_transfer_action": request.Node.StringConfig("post_transfer_action"),
			"ringtone":             request.Node.StringConfig("ringtone"),
			"transfer_delay":       request.Node.StringConfig("transfer_delay"),
		},
	})
	return Result{Terminal: true, ResponseText: transferMessage}, err
}
