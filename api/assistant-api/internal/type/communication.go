// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_type

import (
	"context"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"

	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

type InternalCaller interface {

	// integration calling // router
	IntegrationCaller() integration_client.IntegrationServiceClient

	// for calling vault
	VaultCaller() web_client.VaultClient

	// for calling endpoint
	DeploymentCaller() endpoint_client.DeploymentServiceClient
}

type Communication interface {

	// llm callback
	Callback

	//caller
	InternalCaller

	// authentication
	Auth() *types.Authentication

	// phone, debugger, sdk etc
	GetSource() utils.RapidaSource

	// current assistant
	Assistant() (*internal_assistant_entity.Assistant, error)

	//
	GetMode() type_enums.MessageMode

	// current conversation
	Conversation() (*internal_conversation_entity.AssistantConversation, error)

	// later will create an interface to move all the conversation
	// idea is have custom history maintainer eg: database, inmemory
	// local managing the histories for given conversation
	GetHistories() []MessagePacket

	// metadata management
	Metadata() map[string]interface{}
	GetArgs() map[string]interface{}
	GetOptions() utils.Option
	GetKnowledge(ctx context.Context, knowledgeId uint64) (*internal_knowledge_gorm.Knowledge, error)
	RetrieveToolKnowledge(
		ctx context.Context,
		knowledge *internal_knowledge_gorm.Knowledge,
		conversationMessageId string,
		query string,
		filter map[string]interface{},
		kc *KnowledgeRetrieveOption,
	) ([]KnowledgeContextResult, error)

	// IsConditionAllowed checks if a condition is allowed based on the provided options and key
	IsConditionAllowed(opts utils.Option, key string) bool
}
