// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_local

import (
	"context"
	"fmt"
	"strings"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	internal_tool "github.com/rapidaai/api/assistant-api/internal/tool/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
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

func (tc *knowledgeRetrievalToolCaller) argument(input map[string]interface{}) (*string, map[string]interface{}, error) {
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

func (t *knowledgeRetrievalToolCaller) Call(ctx context.Context, contextID, toolId string, args map[string]interface{}, communication internal_type.Communication) {
	communication.OnPacket(ctx, internal_type.LLMToolCallPacket{
		ToolID: toolId, Name: t.Name(), ContextID: contextID, Arguments: internal_tool.StringifyArgs(args),
	})
	in, v, err := t.argument(args)
	if err != nil || in == nil {
		communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
			ToolID: toolId, Name: t.Name(), ContextID: contextID,
			Result: internal_tool.ErrorResult("Required argument is missing or query, context is missing from argument list"),
		})
		return
	}
	knowledges, err := communication.RetrieveToolKnowledge(ctx,
		t.knowledge, contextID, *in, v, &internal_type.KnowledgeRetrieveOption{
			EmbeddingProviderCredential: t.providerCredential,
			RetrievalMethod:             t.searchType,
			TopK:                        t.topK,
			ScoreThreshold:              float32(t.scoreThreshold),
		})
	if len(knowledges) == 0 || err != nil {
		communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
			ToolID: toolId, Name: t.Name(), ContextID: contextID,
			Result: internal_tool.ErrorResult("Not able to find anything in knowledge from given documents."),
		})
		return
	}
	var contextBuilder strings.Builder
	for _, knowledge := range knowledges {
		contextBuilder.WriteString(knowledge.Content)
		contextBuilder.WriteString("\n")
	}
	communication.OnPacket(ctx, internal_type.LLMToolResultPacket{
		ToolID: toolId, Name: t.Name(), ContextID: contextID,
		Result: internal_tool.Result(contextBuilder.String(), true),
	})
}

func NewKnowledgeRetrievalToolCaller(
	ctx context.Context,
	logger commons.Logger,
	toolOptions *internal_assistant_entity.AssistantTool,
	communication internal_type.Communication,
) (internal_tool.ToolCaller, error) {
	opts := toolOptions.GetOptions()
	searchType, err := opts.GetString("tool.search_type")
	if err != nil {
		return nil, fmt.Errorf("tool.search_type is required: %v", err)
	}

	topK, err := opts.GetUint32("tool.top_k")
	if err != nil {
		return nil, fmt.Errorf("tool.top_k is required: %v", err)
	}

	scoreThreshold, err := opts.GetFloat64("tool.score_threshold")
	if err != nil {
		return nil, fmt.Errorf("tool.score_threshold is not a valid float: %v", err)
	}

	knowledgeID, err := opts.GetUint64("tool.knowledge_id")
	if err != nil {
		return nil, fmt.Errorf("tool.knowledge_id is not a valid number: %v", err)
	}

	knowledge, err := communication.GetKnowledge(ctx, knowledgeID)
	if err != nil {
		logger.Errorf("error while getting knowledge %v", err)
		return nil, err
	}

	credentialId, err := knowledge.GetOptions().GetUint64("rapida.credential_id")
	if err != nil {
		logger.Errorf("error while getting knowledge credentials, check the setup %v", err)
		return nil, err
	}
	providerCredential, err := communication.
		VaultCaller().
		GetCredential(
			ctx,
			communication.Auth(),
			credentialId,
		)

	if err != nil {
		logger.Errorf("error while getting provider model credentials %v for embedding provide model id %s", err, knowledge.EmbeddingModelProviderName)
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
