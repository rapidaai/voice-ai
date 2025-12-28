// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter_requests

import (
	"context"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_gorm "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	internal_adapter_tracing "github.com/rapidaai/api/assistant-api/internal/telemetry"
	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type InternalCaller interface {

	// integration calling // router
	IntegrationCaller() integration_client.IntegrationServiceClient

	// for calling vault
	VaultCaller() web_client.VaultClient

	// for calling endpoint
	DeploymentCaller() endpoint_client.DeploymentServiceClient
}

type Notifier interface {
	// Notifier defines methods for sending notifications related to conversation actions, messages, and events.
	Notify(
		ctx context.Context,
		actionData interface{},
	) error
}

type Logger interface {
	GetConversationLogs() []*protos.Message
	CreateConversationMessageLog(messageid string, in, out *types.Message, metrics []*types.Metric) error
	CreateConversationToolLog(messageid string, in, out map[string]interface{}, metrics []*types.Metric) error
	CreateWebhookLog(
		webhookID uint64, httpUrl, httpMethod, event string,
		responseStatus int64,
		timeTaken int64,
		retryCount uint32,
		status type_enums.RecordState,
		request, response []byte) error
	CreateToolLog(
		toolId uint64,
		messageId string,
		toolName string,
		executionMethod string,
		status type_enums.RecordState,
		timeTaken int64,
		request, response []byte,
	) error

	CreateConversationRecording(
		body []byte,
	) error
}

type Communication interface {

	// stream notification
	Notifier

	// llm callback
	LLMCallback

	//caller
	InternalCaller

	// logging everything
	Logger

	// background context
	Context() context.Context

	// authentication
	Auth() types.SimplePrinciple

	// phone, debugger, sdk etc
	Source() utils.RapidaSource

	// for tracing
	Tracer() internal_adapter_tracing.VoiceAgentTracer

	// current assistant
	Assistant() *internal_assistant_entity.Assistant

	// deployment behavior
	GetBehavior() (*internal_assistant_entity.AssistantDeploymentBehavior, error)

	// current conversation
	Conversation() *internal_conversation_gorm.AssistantConversation

	// later will create an interface to move all the conversation
	// idea is have custom history maintainer eg: database, inmemory
	// local managing the histories for given conversation
	GetHistories() []*types.Message

	// metadata management
	GetMetadata() map[string]interface{}
	GetArgs() map[string]interface{}
	GetOptions() map[string]interface{}

	//
	GetKnowledge(knowledgeId uint64) (*internal_knowledge_gorm.Knowledge, error)

	RetriveToolKnowledge(
		knowledge *internal_knowledge_gorm.Knowledge,
		conversationMessageId string,
		query string,
		filter map[string]interface{},
		kc *KnowledgeRetriveOption,
	) ([]KnowledgeContextResult, error)
}
