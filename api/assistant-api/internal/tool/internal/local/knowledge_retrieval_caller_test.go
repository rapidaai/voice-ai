// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tool_local

import (
	"context"
	"errors"
	"fmt"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type knowledgeCallerLogger struct {
	commons.Logger
	errorMessage string
}

func (logger *knowledgeCallerLogger) Errorf(format string, args ...interface{}) {
	logger.errorMessage = fmt.Sprintf(format, args...)
}

type knowledgeCallerVault struct {
	err error
}

func (vault *knowledgeCallerVault) GetCredential(context.Context, *types.Authentication, uint64) (*protos.VaultCredential, error) {
	return nil, vault.err
}

func (vault *knowledgeCallerVault) GetOauth2Credential(context.Context, *types.Authentication, uint64) (*protos.VaultCredential, error) {
	return nil, vault.err
}

type knowledgeCallerCommunication struct {
	internal_type.Communication
	knowledge *internal_knowledge_gorm.Knowledge
	vault     web_client.VaultClient
}

func (communication *knowledgeCallerCommunication) GetKnowledge(context.Context, uint64) (*internal_knowledge_gorm.Knowledge, error) {
	return communication.knowledge, nil
}

func (communication *knowledgeCallerCommunication) VaultCaller() web_client.VaultClient {
	return communication.vault
}

func (communication *knowledgeCallerCommunication) Auth() *types.Authentication {
	return &types.Authentication{}
}

func TestNewKnowledgeRetrievalToolCaller_LogsEmbeddingProviderOnCredentialFailure(t *testing.T) {
	vaultErr := errors.New("vault unavailable")
	logger := &knowledgeCallerLogger{}
	communication := &knowledgeCallerCommunication{
		knowledge: &internal_knowledge_gorm.Knowledge{
			EmbeddingModelProviderName: "openai",
			KnowledgeEmbeddingModelOptions: []*internal_knowledge_gorm.KnowledgeEmbeddingModelOption{
				{Metadata: *gorm_model.NewMetadata("rapida.credential_id", 2)},
			},
		},
		vault: &knowledgeCallerVault{err: vaultErr},
	}
	toolOptions := &internal_assistant_entity.AssistantTool{
		ExecutionOptions: []*internal_assistant_entity.AssistantToolOption{
			{Metadata: *gorm_model.NewMetadata("tool.search_type", "semantic")},
			{Metadata: *gorm_model.NewMetadata("tool.top_k", 3)},
			{Metadata: *gorm_model.NewMetadata("tool.score_threshold", 0.5)},
			{Metadata: *gorm_model.NewMetadata("tool.knowledge_id", 1)},
		},
	}

	_, err := NewKnowledgeRetrievalToolCaller(context.Background(), logger, toolOptions, communication)

	require.ErrorIs(t, err, vaultErr)
	assert.Equal(t, "error while getting provider model credentials vault unavailable for embedding provide model id openai", logger.errorMessage)
}
