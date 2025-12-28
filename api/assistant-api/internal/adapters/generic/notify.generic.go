// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter_request_generic

import (
	"context"

	internal_adapter_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

// sendMessage is a helper function that centralizes the logic for sending a response via the stream.
func (n *GenericRequestor) sendMessage(ctx context.Context, response *protos.AssistantMessagingResponse) error {
	if err := n.Streamer().Send(response); err != nil {
		return nil
	}
	return nil
}

// notify to websocket when any event
func (n *GenericRequestor) Notify(
	ctx context.Context,
	actionData interface{},
) error {
	// Process actions based on their type
	ctx, span, _ := n.Tracer().StartSpan(ctx, utils.AssistantNotifyStage)
	defer span.EndSpan(ctx, utils.AssistantNotifyStage)
	//
	switch actionData := actionData.(type) {
	case *protos.AssistantConversationUserMessage:
		utils.Go(ctx, func() {
			n.sendMessage(ctx, &protos.AssistantMessagingResponse{
				Code:    200,
				Success: true,
				Data: &protos.AssistantMessagingResponse_User{
					User: actionData,
				},
			})
		})
		span.AddAttributes(ctx,
			internal_adapter_telemetry.KV{
				K: "actor", V: internal_adapter_telemetry.StringValue("user"),
			},
			internal_adapter_telemetry.KV{
				K: "activity", V: internal_adapter_telemetry.StringValue("user_speaking"),
			},
			internal_adapter_telemetry.KV{
				K: "completed", V: internal_adapter_telemetry.BoolValue(actionData.GetCompleted()),
			},
			internal_adapter_telemetry.KV{
				K: "messageId", V: internal_adapter_telemetry.StringValue(actionData.Id),
			})
		switch lt := actionData.Message.(type) {
		case *protos.AssistantConversationUserMessage_Text:
			span.AddAttributes(ctx,
				internal_adapter_telemetry.KV{
					K: "notificaiton_type", V: internal_adapter_telemetry.StringValue("text"),
				},
				internal_adapter_telemetry.KV{
					K: "content_length", V: internal_adapter_telemetry.IntValue(len(lt.Text.Content)),
				},
				internal_adapter_telemetry.KV{
					K: "content", V: internal_adapter_telemetry.StringValue(lt.Text.Content),
				})
		}

		return nil
	case *protos.AssistantConversationAssistantMessage:
		err := n.sendMessage(ctx, &protos.AssistantMessagingResponse{
			Code:    200,
			Success: true,
			Data: &protos.AssistantMessagingResponse_Assistant{
				Assistant: actionData,
			},
		})
		span.AddAttributes(ctx,
			internal_adapter_telemetry.KV{
				K: "actor", V: internal_adapter_telemetry.StringValue("assistant"),
			}, internal_adapter_telemetry.KV{
				K: "activity", V: internal_adapter_telemetry.StringValue("assistant_speaking"),
			}, internal_adapter_telemetry.KV{
				K: "completed", V: internal_adapter_telemetry.BoolValue(actionData.GetCompleted()),
			}, internal_adapter_telemetry.KV{
				K: "messageId", V: internal_adapter_telemetry.StringValue(actionData.Id),
			})
		switch lt := actionData.Message.(type) {
		case *protos.AssistantConversationAssistantMessage_Audio:
			span.AddAttributes(ctx,
				internal_adapter_telemetry.KV{
					K: "notificaiton_type", V: internal_adapter_telemetry.StringValue("audio"),
				},
				internal_adapter_telemetry.KV{
					K: "content_length", V: internal_adapter_telemetry.IntValue(len(lt.Audio.GetContent())),
				})
		case *protos.AssistantConversationAssistantMessage_Text:
			span.AddAttributes(ctx,
				internal_adapter_telemetry.KV{
					K: "notificaiton_type", V: internal_adapter_telemetry.StringValue("text"),
				},
				internal_adapter_telemetry.KV{
					K: "content_length", V: internal_adapter_telemetry.IntValue(len(lt.Text.Content)),
				},
				internal_adapter_telemetry.KV{
					K: "content", V: internal_adapter_telemetry.StringValue(lt.Text.Content),
				})
		}
		return err
	case *protos.AssistantConversationMessage:
		err := n.sendMessage(ctx, &protos.AssistantMessagingResponse{
			Code:    200,
			Success: true,
			Data: &protos.AssistantMessagingResponse_Message{
				Message: actionData,
			},
		})
		span.AddAttributes(ctx,
			internal_adapter_telemetry.KV{
				K: "actor", V: internal_adapter_telemetry.StringValue("system"),
			}, internal_adapter_telemetry.KV{
				K: "activity", V: internal_adapter_telemetry.StringValue("messaging"),
			}, internal_adapter_telemetry.KV{
				K: "messageId", V: internal_adapter_telemetry.StringValue(actionData.MessageId),
			})
		return err
	case *protos.AssistantConversationInterruption:
		utils.Go(ctx, func() {
			n.sendMessage(ctx, &protos.AssistantMessagingResponse{
				Code:    200,
				Success: true,
				Data: &protos.AssistantMessagingResponse_Interruption{
					Interruption: actionData,
				},
			})
		})
		span.AddAttributes(ctx,
			internal_adapter_telemetry.KV{
				K: "actor", V: internal_adapter_telemetry.StringValue("system"),
			}, internal_adapter_telemetry.KV{
				K: "activity", V: internal_adapter_telemetry.StringValue("interrupting"),
			}, internal_adapter_telemetry.KV{
				K: "messageId", V: internal_adapter_telemetry.StringValue(actionData.Id),
			})

	case *protos.AssistantConversationConfiguration:
		// Handle configuration actions
		utils.Go(ctx, func() {
			n.sendMessage(ctx, &protos.AssistantMessagingResponse{
				Code:    200,
				Success: true,
				Data: &protos.AssistantMessagingResponse_Configuration{
					Configuration: actionData,
				},
			})
		})
		span.AddAttributes(ctx,
			internal_adapter_telemetry.KV{
				K: "actor", V: internal_adapter_telemetry.StringValue("system"),
			},

			internal_adapter_telemetry.KV{
				K: "activity", V: internal_adapter_telemetry.StringValue("assistant_configuration"),
			},
		)
	case *protos.AssistantMessagingResponse_Action:
		utils.Go(ctx, func() {
			n.sendMessage(ctx, &protos.AssistantMessagingResponse{
				Code:    200,
				Success: true,
				Data:    actionData,
			})
		})
		span.AddAttributes(
			ctx,
			internal_adapter_telemetry.KV{
				K: "actor", V: internal_adapter_telemetry.StringValue("action"),
			},
			internal_adapter_telemetry.KV{
				K: "activity", V: internal_adapter_telemetry.StringValue("assistant_disconnect"),
			},
		)
	default:
		// Log and return an error for unsupported actions
	}

	return nil
}
