// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"
	"fmt"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (deb *genericRequestor) setAuth(auth *types.Authentication) {
	deb.auth = auth
}

// setMetadata updates in-memory metadata and records changed keys for persistence.
func (tc *genericRequestor) setMetadata(mt map[string]interface{}) {
	if len(mt) == 0 {
		return
	}
	modified := make(map[string]interface{})
	for k, v := range mt {
		vl, ok := tc.metadata[k]
		if ok && vl == v {
			continue
		}
		tc.metadata[k] = v
		modified[k] = v
	}
	if len(modified) == 0 {
		return
	}
	if tc.observabilityRecorder == nil || tc.assistant == nil || tc.assistantConversation == nil {
		return
	}
	// Conversation metadata persistence is owned by the observability metadata collector.
	metadataList := make([]*protos.Metadata, 0, len(modified))
	for key, value := range modified {
		metadataList = append(metadataList, &protos.Metadata{Key: key, Value: metadataValueString(value)})
	}
	if err := tc.observabilityRecorder.Record(context.Background(),
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: tc.assistant.Id},
			ConversationID: tc.assistantConversation.Id,
		},
		observability.RecordMetadata{Metadata: metadataList},
	); err != nil && tc.logger != nil {
		tc.logger.Debugf("error while recording conversation metadata %+v", err)
	}
}

// metadataValueString converts dynamic metadata values to their stored string form.
func metadataValueString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// setArguments merges arguments into in-memory state and persists asynchronously.
func (tc *genericRequestor) setArguments(args map[string]interface{}) {
	if len(args) == 0 {
		return
	}
	tc.args = utils.MergeMaps(tc.args, args)
	assistant, err := tc.Assistant()
	if err != nil {
		return
	}
	conversation, err := tc.Conversation()
	if err != nil {
		return
	}
	// Argument writes are async so request handling is not blocked by DB latency.
	utils.Go(context.Background(), func() {
		dbCtx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
		defer cancel()
		if _, err := tc.conversationService.CreateOrUpdateConversationArgument(
			dbCtx, tc.auth, assistant.Id, conversation.Id, args,
		); err != nil {
			tc.OnPacket(context.Background(), internal_type.ObservabilityLogRecordPacket{
				ContextID: tc.GetID(),
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "conversation arguments persistence failed",
					Attributes: observability.Attributes{
						"component":       observability.ComponentConversation.String(),
						"operation":       "persist_arguments",
						"context_id":      tc.GetID(),
						"assistant_id":    fmt.Sprintf("%d", assistant.Id),
						"conversation_id": fmt.Sprintf("%d", conversation.Id),
						"argument_count":  fmt.Sprintf("%d", len(args)),
						"error":           err.Error(),
						"error_type":      fmt.Sprintf("%T", err),
					},
				},
			})
		}
	})
}

// setOptions merges options into in-memory state and persists asynchronously.
func (tc *genericRequestor) setOptions(opts map[string]interface{}) {
	if len(opts) == 0 {
		return
	}
	tc.options = utils.MergeMaps(tc.options, opts)
	assistant, err := tc.Assistant()
	if err != nil {
		return
	}
	conversation, err := tc.Conversation()
	if err != nil {
		return
	}
	// Option writes are async so request handling is not blocked by DB latency.
	utils.Go(context.Background(), func() {
		dbCtx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
		defer cancel()
		if _, err := tc.conversationService.CreateOrUpdateConversationOption(
			dbCtx, tc.auth, assistant.Id, conversation.Id, opts,
		); err != nil {
			tc.OnPacket(context.Background(), internal_type.ObservabilityLogRecordPacket{
				ContextID: tc.GetID(),
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "conversation options persistence failed",
					Attributes: observability.Attributes{
						"component":       observability.ComponentConversation.String(),
						"operation":       "persist_options",
						"context_id":      tc.GetID(),
						"assistant_id":    fmt.Sprintf("%d", assistant.Id),
						"conversation_id": fmt.Sprintf("%d", conversation.Id),
						"option_count":    fmt.Sprintf("%d", len(opts)),
						"error":           err.Error(),
						"error_type":      fmt.Sprintf("%T", err),
					},
				},
			})
		}
	})
}
