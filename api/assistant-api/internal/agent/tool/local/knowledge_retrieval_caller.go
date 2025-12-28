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
	"strings"
	"time"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type knowledgeRetrievalToolCaller struct {
	toolCaller
	searchType         string
	topK               uint32
	scoreThreshold     float64
	knowledge          *internal_knowledge_gorm.Knowledge
	providerCredential *protos.VaultCredential
}

func (tc *knowledgeRetrievalToolCaller) argument(args string) (*string, map[string]interface{}, error) {
	var input map[string]interface{}
	err := json.Unmarshal([]byte(args), &input)
	if err != nil {
		tc.logger.Debugf("illegal input from llm check and pushing the llm response as incomplete %v", args)
		return nil, nil, err
	}
	var queryOrContext string
	if query, ok := input["query"].(string); ok {
		queryOrContext = query
	} else if context, ok := input["context"].(string); ok {
		queryOrContext = context
	} else {
		return nil, nil, fmt.Errorf("neither query nor context found or not a string in input")
	}
	return utils.Ptr(queryOrContext), input, nil
}
func (afkTool *knowledgeRetrievalToolCaller) Call(
	ctx context.Context,
	messageId string,
	args string,
	communication internal_adapter_requests.Communication,
) (map[string]interface{}, []*types.Metric) {
	start := time.Now()
	metrics := make([]*types.Metric, 0)
	in, v, err := afkTool.argument(args)

	var result map[string]interface{}
	var contextString string

	if err != nil || in == nil {
		result = afkTool.Result("Required argument is missing or query, context is missing from argument list", false)
		metrics = append(metrics, types.NewMetric("retrieval_status", type_enums.RECORD_FAILED.String(), utils.Ptr("status of tool.retrieval_status")))
	} else {
		knowledges, err := communication.RetriveToolKnowledge(
			afkTool.knowledge,
			messageId,
			*in,
			v,
			&internal_adapter_requests.KnowledgeRetriveOption{
				EmbeddingProviderCredential: afkTool.providerCredential,
				RetrievalMethod:             afkTool.searchType,
				TopK:                        afkTool.topK,
				ScoreThreshold:              float32(afkTool.scoreThreshold),
			})

		if len(knowledges) == 0 || err != nil {
			result = afkTool.Result("Not able to find anything in knowledge from given documents.", true)
			metrics = append(metrics, types.NewMetric("retrieval_status", type_enums.RECORD_COMPLETE.String(), utils.Ptr("status of tool.retrieval_status")))
		} else {
			var contextTemplateBuilder strings.Builder
			for _, knowledge := range knowledges {
				contextTemplateBuilder.WriteString(knowledge.Content)
				contextTemplateBuilder.WriteString("\n")
			}
			contextString = contextTemplateBuilder.String()
			result = afkTool.Result(contextString, true)
			metrics = append(metrics, types.NewMetric("retrieval_status", type_enums.RECORD_COMPLETE.String(), utils.Ptr("status of tool.retrieval_status")))
		}
	}

	metrics = append(metrics, types.NewTimeTakenMetric(time.Since(start)))
	return result, metrics
}

func NewKnowledgeRetrievalToolCaller(
	logger commons.Logger,
	toolOptions *internal_assistant_entity.AssistantTool,
	communcation internal_adapter_requests.Communication,
) (ToolCaller, error) {
	opts := toolOptions.GetOptions()
	searchType, err := opts.GetString("tool.search_type")
	if err != nil {
		return nil, fmt.Errorf("tool.search_type is not a recognized type, got %T", err)
	}

	topK, err := opts.GetUint32("tool.top_k")
	if err != nil {
		return nil, fmt.Errorf("tool.top_k is not a recognized type, got %T", err)
	}

	scoreThreshold, err := opts.GetFloat64("tool.score_threshold")
	if err != nil {
		return nil, fmt.Errorf("tool.score_threshold is not a valid float: %v", err)
	}

	knowledgeID, err := opts.GetUint64("tool.knowledge_id")
	if err != nil {
		return nil, fmt.Errorf("tool.knowledge_id is not a valid number: %v", err)
	}

	knowledge, err := communcation.GetKnowledge(knowledgeID)
	if err != nil {
		logger.Errorf("error while getting knowledge %v", err)
		return nil, err
	}

	credentialId, err := knowledge.GetOptions().GetUint64("rapida.credential_id")
	if err != nil {
		logger.Errorf("error while getting knowledge credentials, check the setup %v", err)
		return nil, err
	}
	providerCredential, err := communcation.
		VaultCaller().
		GetCredential(
			communcation.Context(),
			communcation.Auth(),
			credentialId,
		)

	if err != nil {
		logger.Errorf("error while getting provider model credentials %v for embedding provide model id %d", err, knowledge.EmbeddingModelProviderName)
		return nil, err
	}
	return &knowledgeRetrievalToolCaller{
		toolCaller: toolCaller{
			logger:      logger,
			toolOptions: toolOptions,
		},
		searchType:         searchType,
		topK:               topK,
		scoreThreshold:     scoreThreshold,
		providerCredential: providerCredential,
		knowledge:          knowledge,
	}, nil
}
