// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"
	"errors"
	"time"

	"github.com/rapidaai/api/assistant-api/config"
	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	adapter_router "github.com/rapidaai/api/assistant-api/internal/adapters/router"
	"github.com/rapidaai/protos"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"

	internal_agent_embeddings "github.com/rapidaai/api/assistant-api/internal/agent/embedding"
	internal_agent_rerankers "github.com/rapidaai/api/assistant-api/internal/agent/reranker"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	internal_input_normalizer "github.com/rapidaai/api/assistant-api/internal/normalizer/input"
	internal_output_normalizer "github.com/rapidaai/api/assistant-api/internal/normalizer/output"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	internal_knowledge_service "github.com/rapidaai/api/assistant-api/internal/services/knowledge"
	"github.com/rapidaai/api/assistant-api/internal/watchdog"
	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	web_client "github.com/rapidaai/pkg/clients/web"

	//
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

const (
	dbWriteTimeout     = 5 * time.Second
	recordingTimeout   = 3*storages.FileWriteTimeout + dbWriteTimeout
	connectDeadline    = 30 * time.Second
	disconnectDeadline = 30 * time.Second
)

var (
	errDeploymentNotEnabled = errors.New("deployment is not enabled for source")
)

type genericRequestor struct {
	logger   commons.Logger
	config   *config.AssistantConfig
	source   utils.RapidaSource
	auth     *types.Authentication
	streamer internal_type.Streamer

	// service
	assistantService     internal_services.AssistantService
	conversationService  internal_services.AssistantConversationService
	httpLogService       internal_services.AssistantHTTPLogService
	knowledgeService     internal_services.KnowledgeService
	assistantToolService internal_services.AssistantToolService

	//
	postgres      connectors.PostgresConnector
	opensearch    connectors.OpenSearchConnector
	vectordb      connectors.VectorConnector
	queryEmbedder internal_agent_embeddings.QueryEmbedding
	textReranker  internal_agent_rerankers.TextReranking

	observabilityRecorder observability.Recorder
	rapidaClient          *rapida_client.RapidaClient

	// interaction/session lifecycle owners.
	messageLifecycle adapter_lifecycle.MessageLifecycle
	sessionLifecycle adapter_lifecycle.SessionLifecycle
	dispatchRoute    *adapter_router.DispatchRoute

	// listening
	speechToTextTransformer internal_type.SpeechToTextTransformer
	textToSpeechTransformer internal_type.TextToSpeechTransformer

	// audio intelligence

	// output preprocessor + TTS
	inputNormalizer  internal_type.PacketNormalizer
	outputNormalizer internal_type.PacketNormalizer

	// executor
	assistantAnalyseExecutors     []internal_type.AnalysisExecutor
	artifactPushExecutors         []internal_type.ArtifactPushExecutor
	conversationRecordingExecutor internal_type.ConversationRecordingExecutor
	authenticationExecutor        internal_type.AuthenticationExecutor
	assistantExecutor             internal_type.LLMExecutor
	endOfSpeechExecutor           internal_type.EndOfSpeechExecutor
	denoiserExecutor              internal_type.VoiceDenoiserExecutor
	vadExecutor                   internal_type.VoiceActivityDetectorExecutor

	// states
	assistant             *internal_assistant_entity.Assistant
	assistantConversation *internal_conversation_entity.AssistantConversation
	histories             []internal_type.MessagePacket

	args     map[string]interface{}
	metadata map[string]interface{}
	metrics  map[string]*protos.Metric
	options  map[string]interface{}

	// experience
	idleTimeoutWatchdog   *watchdog.IdleTimeoutWatchdog
	unclearInputWatchdog  *watchdog.UnclearInputWatchdog
	ttsCompletionWatchdog *watchdog.TTSCompletionWatchdog
	maxSessionWatchdog    *watchdog.MaxSessionWatchdog

	// sessionCtx is the adapter-owned lifecycle context. Outlives the gRPC stream.
	// cancelSession is invoked exactly once, by HandleFinalizationCompleted, after
	// the disconnect chain has fully drained. No other call site should invoke it.
	sessionCtx    context.Context
	cancelSession context.CancelFunc
	// channel registry with semantic names.
	channels adapter_channel.RequestorChannelBus
}

func NewGenericRequestor(
	ctx context.Context,
	config *config.AssistantConfig,
	rapidaClient *rapida_client.RapidaClient,
	logger commons.Logger, source utils.RapidaSource,
	postgres connectors.PostgresConnector, opensearch connectors.OpenSearchConnector,
	redis connectors.RedisConnector, storage storages.Storage, streamer internal_type.Streamer,
	observer observability.Recorder,
) *genericRequestor {
	sessionCtx, cancelSession := context.WithCancel(context.Background())
	channels := adapter_channel.NewRequestorChannels()
	gr := &genericRequestor{
		logger:   logger,
		config:   config,
		source:   source,
		streamer: streamer,
		// services
		assistantService:     internal_assistant_service.NewAssistantService(config, logger, postgres, opensearch),
		knowledgeService:     internal_knowledge_service.NewKnowledgeService(config, logger, postgres, storage),
		conversationService:  internal_assistant_service.NewAssistantConversationService(logger, postgres, storage),
		httpLogService:       internal_assistant_service.NewAssistantHTTPLogService(logger, postgres, storage),
		assistantToolService: internal_assistant_service.NewAssistantToolService(logger, postgres, storage),
		//

		postgres:      postgres,
		opensearch:    opensearch,
		vectordb:      opensearch,
		queryEmbedder: internal_agent_embeddings.NewQueryEmbedding(logger, config, redis),
		textReranker:  internal_agent_rerankers.NewTextReranker(logger, config, redis),

		rapidaClient: rapidaClient,

		observabilityRecorder: observer,

		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycle(),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), channels),

		inputNormalizer:  internal_input_normalizer.NewInputNormalizer(logger),
		outputNormalizer: internal_output_normalizer.NewOutputNormalizer(logger),

		//
		histories:                 make([]internal_type.MessagePacket, 0),
		metadata:                  make(map[string]interface{}),
		metrics:                   make(map[string]*protos.Metric),
		args:                      make(map[string]interface{}),
		options:                   make(map[string]interface{}),
		assistantAnalyseExecutors: make([]internal_type.AnalysisExecutor, 0),
		artifactPushExecutors:     make([]internal_type.ArtifactPushExecutor, 0),
		sessionCtx:                sessionCtx,
		cancelSession:             cancelSession,
		channels:                  channels,
	}

	gr.idleTimeoutWatchdog = watchdog.NewIdleTimeoutWatchdog(
		watchdog.WithOnPacket(gr.OnPacket),
		watchdog.WithPacketContext(sessionCtx),
	)
	gr.unclearInputWatchdog = watchdog.NewUnclearInputWatchdog(
		watchdog.WithOnPacket(gr.OnPacket),
		watchdog.WithPacketContext(sessionCtx),
	)
	gr.ttsCompletionWatchdog = watchdog.NewTTSCompletionWatchdog(
		watchdog.WithOnPacket(gr.OnPacket),
		watchdog.WithPacketContext(sessionCtx),
		watchdog.WithGracePeriod(300*time.Millisecond),
	)
	gr.maxSessionWatchdog = watchdog.NewMaxSessionWatchdog(
		watchdog.WithOnPacket(gr.OnPacket),
		watchdog.WithPacketContext(sessionCtx),
	)

	go gr.runBootstrapDispatcher(sessionCtx)
	go gr.runCriticalDispatcher(sessionCtx)
	go gr.runOutputDispatcher(sessionCtx)
	go gr.runDataDispatcher(sessionCtx)

	return gr
}

// GetSource implements internal_adapter_requests.Messaging.
func (dm *genericRequestor) GetSource() utils.RapidaSource {
	return dm.source
}

func (gr *genericRequestor) getAssistantConversation(ctx context.Context, auth *types.Authentication, assistantId uint64, assistantConversationId uint64) (*internal_conversation_entity.AssistantConversation, error) {
	return gr.conversationService.GetConversation(ctx, auth, assistantId, assistantConversationId, &internal_services.GetConversationOption{
		InjectContext:  true,
		InjectArgument: true,
		InjectMetadata: true,
		InjectOption:   true,
		InjectMetric:   false},
	)
}

func (talking *genericRequestor) BeginConversation(ctx context.Context, assistant *internal_assistant_entity.Assistant, direction type_enums.ConversationDirection, config *protos.ConversationInitialization) error {
	talking.assistant = assistant
	conversation, err := talking.conversationService.CreateConversation(ctx, talking.Auth(), talking.identifier(config), assistant.Id, assistant.AssistantProviderId, direction, talking.GetSource())
	if err != nil {
		return err
	}
	talking.assistantConversation = conversation
	if arguments, err := utils.AnyMapToInterfaceMap(config.GetArgs()); err == nil {
		talking.setArguments(arguments)
	}
	if options, err := utils.AnyMapToInterfaceMap(config.GetOptions()); err == nil {
		talking.setOptions(options)
	}
	if metadata, err := utils.AnyMapToInterfaceMap(config.GetMetadata()); err == nil {
		talking.setMetadata(metadata)
	}
	return err
}

func (talking *genericRequestor) ResumeConversation(ctx context.Context, assistant *internal_assistant_entity.Assistant, config *protos.ConversationInitialization) error {
	talking.assistant = assistant
	conversation, err := talking.getAssistantConversation(ctx, talking.Auth(), assistant.Id, config.GetAssistantConversationId())
	if err != nil {
		return err
	}
	talking.assistantConversation = conversation
	talking.args = conversation.GetArguments()
	talking.options = conversation.GetOptions()
	talking.metadata = conversation.GetMetadatas()
	talking.metrics = make(map[string]*protos.Metric)
	for _, metric := range conversation.Metrics {
		if metric == nil {
			continue
		}
		talking.metrics[metric.Name] = &protos.Metric{
			Name:        metric.Name,
			Value:       metric.Value,
			Description: metric.Description,
		}
	}
	if extra, err := utils.AnyMapToInterfaceMap(config.GetMetadata()); err == nil {
		talking.setMetadata(extra)
	}
	return nil
}

func (talking *genericRequestor) IntegrationCaller() integration_client.IntegrationServiceClient {
	return talking.rapidaClient.Integration

}

func (talking *genericRequestor) VaultCaller() web_client.VaultClient {
	return talking.rapidaClient.Vault
}

func (talking *genericRequestor) DeploymentCaller() endpoint_client.DeploymentServiceClient {
	return talking.rapidaClient.Deployment
}

func (talking *genericRequestor) GetKnowledge(ctx context.Context, knowledgeId uint64) (*internal_knowledge_gorm.Knowledge, error) {
	return talking.knowledgeService.Get(ctx, talking.auth, knowledgeId)
}

func (gr *genericRequestor) GetArgs() map[string]interface{} {
	return gr.args
}

func (gr *genericRequestor) GetOptions() utils.Option {
	return gr.options
}

func (dm *genericRequestor) GetHistories() []internal_type.MessagePacket {
	return dm.histories
}

// =============================================================================
// Interaction state methods — inline replacement for the former Messaging wrapper
// =============================================================================

func (r *genericRequestor) GetID() string {
	return r.messageLifecycle.ContextID()
}

// GetMode returns the current stream mode (text or audio).
func (r *genericRequestor) GetMode() type_enums.MessageMode {
	return r.messageLifecycle.Mode()
}

// SwitchMode sets the stream mode.
func (r *genericRequestor) SwitchMode(mm type_enums.MessageMode) {
	r.messageLifecycle.SetMode(mm)
}

func (r *genericRequestor) canSwitchSession() bool {
	return r.sessionLifecycle.CanBe(adapter_lifecycle.EventSwitchRequested)
}
