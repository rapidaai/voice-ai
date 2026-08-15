// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	internal_agent_embeddings "github.com/rapidaai/api/assistant-api/internal/agent/embedding"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/connectors"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	defaultTopK           = 4
	defaultScoreThreshold = 0.5

	retrievalHybrid   = "hybrid"
	retrievalSemantic = "semantic"
	retrievalText     = "text"
)

func (kr *genericRequestor) RetrieveToolKnowledge(ctx context.Context, knowledge *internal_knowledge_gorm.Knowledge, messageId string, query string, filter map[string]interface{}, kc *internal_type.KnowledgeRetrieveOption) ([]internal_type.KnowledgeContextResult, error) {
	start := time.Now()
	if messageId == "" {
		messageId = kr.GetID()
	}

	knowledgeIDStr := fmt.Sprintf("%d", knowledge.Id)
	kr.OnPacket(ctx, internal_type.ObservabilityEventRecordPacket{
		ContextID: messageId,
		Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
		Record: observability.NewMessageRecord(messageId, observability.ComponentTool, observability.ToolCallStarted, observability.MessageRoleAssistant, observability.Attributes{
			"name":             "knowledge",
			"knowledge_id":     knowledgeIDStr,
			"operation":        "retrieving",
			"method":           kc.RetrievalMethod,
			"top_k":            fmt.Sprintf("%d", kc.TopK),
			"query_char_count": fmt.Sprintf("%d", len(query)),
		}),
	})

	result, err := kr.retrieve(ctx, knowledge, query, filter, kc)

	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		kr.OnPacket(ctx,
			internal_type.ObservabilityEventRecordPacket{
				ContextID: messageId,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMessageRecord(messageId, observability.ComponentTool, observability.ToolCallFailed, observability.MessageRoleAssistant, observability.Attributes{
					"name":         "knowledge",
					"knowledge_id": knowledgeIDStr,
					"operation":    "retrieval",
					"error":        err.Error(),
				}),
			})
	} else {
		topScore := 0.0
		if len(result) > 0 {
			topScore = result[0].Score
		}
		kr.OnPacket(ctx,
			internal_type.ObservabilityEventRecordPacket{
				ContextID: messageId,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMessageRecord(messageId, observability.ComponentTool, observability.ToolCallCompleted, observability.MessageRoleAssistant, observability.Attributes{
					"name":         "knowledge",
					"knowledge_id": knowledgeIDStr,
					"operation":    "retrieval",
					"method":       kc.RetrievalMethod,
					"result_count": fmt.Sprintf("%d", len(result)),
					"top_score":    fmt.Sprintf("%.4f", topScore),
				}),
			},
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: messageId,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMessageMetricRecord(messageId, observability.MessageRoleAssistant, []*protos.Metric{{
					Name:  observability.MetricKnowledgeLatencyMs,
					Value: fmt.Sprintf("%d", latencyMs),
				}}),
			},
		)
	}

	utils.Go(context.Background(), func() {
		request, _ := json.Marshal(map[string]interface{}{
			"query":  query,
			"filter": filter,
		})
		var response []byte
		status := type_enums.RECORD_COMPLETE
		if err != nil {
			response, _ = json.Marshal(map[string]string{"error": err.Error()})
			status = type_enums.RECORD_FAILED
		} else {
			response, _ = json.Marshal(map[string]interface{}{
				"result": result,
			})
		}
		kr.createKnowledgeLog(
			ctx,
			knowledge.Id,
			kc.RetrievalMethod,
			kc.TopK,
			kc.ScoreThreshold,
			len(result),
			int64(time.Since(start)),
			map[string]string{
				"source":                         "tool",
				"assistantId":                    fmt.Sprintf("%d", kr.assistant.Id),
				"assistantConversationId":        fmt.Sprintf("%d", kr.assistantConversation.Id),
				"assistantConversationMessageId": messageId,
			},
			status,
			request, response,
		)
	})
	return result, err

}

func (kr *genericRequestor) retrieve(ctx context.Context, knowledge *internal_knowledge_gorm.Knowledge, query string, filter map[string]interface{}, kc *internal_type.KnowledgeRetrieveOption) ([]internal_type.KnowledgeContextResult, error) {
	if kr.vectordb == nil {
		return nil, fmt.Errorf("knowledge retrieval is not available: vector database is not configured")
	}
	topK := int(defaultTopK)
	if kc.TopK != 0 {
		topK = int(kc.TopK)
	}
	minScore := float32(defaultScoreThreshold)
	if kc.ScoreThreshold != 0 {
		minScore = float32(kc.ScoreThreshold)
	}
	results := make([]internal_type.KnowledgeContextResult, 0)

	switch kc.RetrievalMethod {
	case "hybrid-search", retrievalHybrid:
		embeddingOpts := &internal_agent_embeddings.TextEmbeddingOption{
			ProviderCredential: kc.EmbeddingProviderCredential,
			ModelProviderName:  knowledge.EmbeddingModelProviderName,
			Options:            knowledge.GetOptions(),
			AdditionalData: map[string]string{
				"knowledge_id": fmt.Sprintf("%d", knowledge.Id),
			},
		}
		embeddings, err := kr.queryEmbedder.TextQueryEmbedding(ctx, kr.Auth(), query, embeddingOpts)
		if err != nil {
			return results, err
		}
		matchedContents, err := kr.vectordb.HybridSearch(ctx,
			knowledge.StorageNamespace,
			query,
			embeddings.Data[len(embeddings.Data)-1].GetEmbedding(),
			filter,
			connectors.NewDefaultVectorSearchOptions(
				connectors.WithMinScore(minScore),
				connectors.WithSource([]string{"text", "document_id", "metadata"}),
				connectors.WithTopK(topK)))
		if err != nil {
			return results, err
		}
		for _, x := range matchedContents {
			source := x["_source"].(map[string]interface{})
			results = append(results, internal_type.KnowledgeContextResult{
				ID:         x["_id"].(string),
				DocumentID: source["document_id"].(string),
				Metadata:   source["metadata"].(map[string]interface{}),
				Content:    source["text"].(string),
				Score:      x["_score"].(float64),
			})
		}
		return results, err

	case "semantic-search", retrievalSemantic:
		embeddings, err := kr.queryEmbedder.TextQueryEmbedding(
			ctx,
			kr.Auth(),
			query, &internal_agent_embeddings.TextEmbeddingOption{
				ProviderCredential: kc.EmbeddingProviderCredential,
				ModelProviderName:  knowledge.EmbeddingModelProviderName,
				Options:            knowledge.GetOptions(),
				AdditionalData: map[string]string{
					"knowledge_id": fmt.Sprintf("%d", knowledge.Id),
				},
			})
		if err != nil {
			return results, err
		}

		matchedContents, err := kr.vectordb.VectorSearch(
			ctx,
			knowledge.StorageNamespace,
			embeddings.Data[len(embeddings.Data)-1].GetEmbedding(),
			filter,
			connectors.NewDefaultVectorSearchOptions(
				connectors.WithSource([]string{"text", "document_id", "metadata"}),
				connectors.WithMinScore(minScore), connectors.WithTopK(topK)),
		)
		if err != nil {
			return results, err
		}

		for _, x := range matchedContents {
			source := x["_source"].(map[string]interface{})
			results = append(results, internal_type.KnowledgeContextResult{
				ID:         x["_id"].(string),
				DocumentID: source["document_id"].(string),
				Metadata:   source["metadata"].(map[string]interface{}),
				Content:    source["text"].(string),
				Score:      x["_score"].(float64),
			})
		}
		return results, err

	case "text-search", retrievalText:
		matchedContents, err := kr.vectordb.TextSearch(
			ctx,
			knowledge.StorageNamespace,
			query,
			filter,
			connectors.NewDefaultVectorSearchOptions(
				connectors.WithSource([]string{"text", "document_id", "metadata"}),
				connectors.WithMinScore(minScore),
				connectors.WithTopK(topK)))
		if err != nil {
			return results, err
		}
		for _, x := range matchedContents {
			source := x["_source"].(map[string]interface{})
			results = append(results, internal_type.KnowledgeContextResult{
				ID:         x["_id"].(string),
				DocumentID: source["document_id"].(string),
				Metadata:   source["metadata"].(map[string]interface{}),
				Content:    source["text"].(string),
				Score:      x["_score"].(float64),
			})
		}
		return results, nil

	default:
		return results, fmt.Errorf("retrieve method is unexpected")
	}
}
