// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter_request_generic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	internal_agent_embeddings "github.com/rapidaai/api/assistant-api/internal/agent/embedding"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	"github.com/rapidaai/pkg/connectors"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

var DEFAULT_TOP_K = 4
var DEFAULT_SCORE_THRESHOLD = 0.5

func (kr *GenericRequestor) RetriveToolKnowledge(
	knowledge *internal_knowledge_gorm.Knowledge,
	messageId string,
	query string,
	filter map[string]interface{},
	kc *internal_adapter_requests.KnowledgeRetriveOption,
) ([]internal_adapter_requests.KnowledgeContextResult, error) {
	start := time.Now()
	result, err := kr.retrive(kr.Context(), knowledge, query, filter, kc)
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
		kr.CreateKnowledgeLog(
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

func (kr *GenericRequestor) retrive(
	ctx context.Context,
	knowledge *internal_knowledge_gorm.Knowledge,
	query string,
	filter map[string]interface{},
	kc *internal_adapter_requests.KnowledgeRetriveOption,
) ([]internal_adapter_requests.KnowledgeContextResult, error) {
	topK := int(DEFAULT_TOP_K)
	if kc.TopK != 0 {
		topK = int(kc.TopK)
	}
	minScore := float32(DEFAULT_SCORE_THRESHOLD)
	if kc.ScoreThreshold != 0 {
		minScore = float32(kc.ScoreThreshold)
	}
	Results := make([]internal_adapter_requests.KnowledgeContextResult, 0)
	//
	switch kc.RetrievalMethod {
	case "hybrid-search", "hybrid":
		embeddingOpts := &internal_agent_embeddings.TextEmbeddingOption{
			ProviderCredential: kc.EmbeddingProviderCredential,
			ModelProviderName:  knowledge.EmbeddingModelProviderName,
			Options:            knowledge.GetOptions(),
			AdditionalData: map[string]string{
				"knowledge_id": fmt.Sprintf("%d", knowledge.Id),
			},
		}
		embeddings, err := kr.queryEmbedder.TextQueryEmbedding(
			ctx,
			kr.Auth(),
			query,
			embeddingOpts,
		)
		if err != nil {
			kr.logger.Errorf("Unable to get query embedding from integration for query %s error %v", query, err)
			return Results, err
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
			kr.logger.Errorf("Unable to get result from the vector dataset for given %s error %v", query, err)
			return Results, err
		}
		for _, x := range matchedContents {
			source := x["_source"].(map[string]interface{})
			Results = append(Results, internal_adapter_requests.KnowledgeContextResult{
				ID:         x["_id"].(string),
				DocumentID: source["document_id"].(string),
				Metadata:   source["metadata"].(map[string]interface{}),
				Content:    source["text"].(string),
				Score:      x["_score"].(float64),
			})
		}
		return Results, err

	case "semantic-search", "semantic":
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
			kr.logger.Errorf("Unable to get query embedding from integration for query %s error %v", query, err)
			return Results, err
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
			kr.logger.Errorf("Unable to get result from the vector dataset for given %s error %v", query, err)
			return Results, err
		}

		for _, x := range matchedContents {
			source := x["_source"].(map[string]interface{})
			Results = append(Results, internal_adapter_requests.KnowledgeContextResult{
				ID:         x["_id"].(string),
				DocumentID: source["document_id"].(string),
				Metadata:   source["metadata"].(map[string]interface{}),
				Content:    source["text"].(string),
				Score:      x["_score"].(float64),
			})
		}
		return Results, err

	case "text-search", "text":
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
			kr.logger.Errorf("Unable to get result from the vector dataset for given %s error %v", query, err)
			return Results, nil
		}
		for _, x := range matchedContents {
			source := x["_source"].(map[string]interface{})
			Results = append(Results, internal_adapter_requests.KnowledgeContextResult{
				ID:         x["_id"].(string),
				DocumentID: source["document_id"].(string),
				Metadata:   source["metadata"].(map[string]interface{}),
				Content:    source["text"].(string),
				Score:      x["_score"].(float64),
			})
		}
		return Results, nil

	default:
		kr.logger.Errorf("retrive method is unexpected")
		return Results, fmt.Errorf("retrive method is unexpected")
	}
}
