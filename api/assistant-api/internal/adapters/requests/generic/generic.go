package internal_adapter_request_generic

import (
	"context"
	"time"

	"github.com/rapidaai/api/assistant-api/config"
	internal_adapter_request_customizers "github.com/rapidaai/api/assistant-api/internal/adapters/requests/customizers"
	internal_adapter_request_streamers "github.com/rapidaai/api/assistant-api/internal/adapters/requests/streamers"
	internal_agent_embeddings "github.com/rapidaai/api/assistant-api/internal/agents/embeddings"
	internal_agent_rerankers "github.com/rapidaai/api/assistant-api/internal/agents/rerankers"
	internal_analyzers "github.com/rapidaai/api/assistant-api/internal/analyzers"
	internal_executors "github.com/rapidaai/api/assistant-api/internal/executors"
	internal_assistant_executors "github.com/rapidaai/api/assistant-api/internal/executors/assistant"
	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	internal_conversation_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/conversations"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/knowledges"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	internal_knowledge_service "github.com/rapidaai/api/assistant-api/internal/services/knowledge"
	internal_synthesizers "github.com/rapidaai/api/assistant-api/internal/synthesizes"
	internal_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	internal_assistant_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry/assistant"
	internal_assistant_telemetry_exporters "github.com/rapidaai/api/assistant-api/internal/telemetry/assistant/exporters"
	internal_transcribes "github.com/rapidaai/api/assistant-api/internal/transcribers"
	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/parsers"

	//
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

type GenericRequestor struct {
	logger   commons.Logger
	config   *config.AssistantConfig
	ctx      context.Context
	source   utils.RapidaSource
	auth     types.SimplePrinciple
	streamer internal_adapter_request_streamers.Streamer

	// service
	assistantService     internal_services.AssistantService
	conversationService  internal_services.AssistantConversationService
	webhookService       internal_services.AssistantWebhookService
	knowledgeService     internal_services.KnowledgeService
	assistantToolService internal_services.AssistantToolService

	//
	opensearch    connectors.OpenSearchConnector
	vectordb      connectors.VectorConnector
	queryEmbedder internal_agent_embeddings.QueryEmbedding
	textReranker  internal_agent_rerankers.TextReranking

	// managing event
	tracer internal_telemetry.VoiceAgentTracer

	// integration client
	integrationClient integration_client.IntegrationServiceClient
	vaultClient       web_client.VaultClient
	deploymentClient  endpoint_client.DeploymentServiceClient

	// io related
	messaging internal_adapter_request_customizers.Messaging

	// listening
	speechToTextTransformer internal_transformers.SpeechToTextTransformer
	audioAnalyzers          []internal_analyzers.AudioAnalyzer
	textAnalyzers           []internal_analyzers.TextAnalyzer

	// speak
	outputAudioTransformer internal_transformers.TextToSpeechTransformer
	//
	transcriber  internal_transcribes.Transcriber
	synthesizers []internal_synthesizers.SentenceSynthesizer

	recorder       internal_adapter_request_customizers.Recorder
	templateParser parsers.StringTemplateParser

	// executor
	assistantExecutor internal_executors.AssistantExecutor
	// states
	assistant             *internal_assistant_gorm.Assistant
	assistantConversation *internal_conversation_gorm.AssistantConversation
	histories             []*types.Message
	metrics               []*types.Metric
	args                  map[string]interface{}
	metadata              map[string]interface{}
	options               map[string]interface{}
	StartedAt             time.Time
}

func NewGenericRequestor(
	ctx context.Context,
	config *config.AssistantConfig,
	logger commons.Logger,
	source utils.RapidaSource,
	postgres connectors.PostgresConnector,
	opensearch connectors.OpenSearchConnector,
	redis connectors.RedisConnector,
	storage storages.Storage,
	streamer internal_adapter_request_streamers.Streamer,
) GenericRequestor {

	return GenericRequestor{
		logger:   logger,
		config:   config,
		ctx:      ctx,
		source:   source,
		streamer: streamer,
		// services
		assistantService:     internal_assistant_service.NewAssistantService(config, logger, postgres, opensearch),
		knowledgeService:     internal_knowledge_service.NewKnowledgeService(config, logger, postgres, storage),
		conversationService:  internal_assistant_service.NewAssistantConversationService(logger, postgres, storage),
		webhookService:       internal_assistant_service.NewAssistantWebhookService(logger, postgres, storage),
		assistantToolService: internal_assistant_service.NewAssistantToolService(logger, postgres, storage),
		//

		opensearch:    opensearch,
		vectordb:      opensearch,
		queryEmbedder: internal_agent_embeddings.NewQueryEmbedding(logger, config, redis),
		textReranker:  internal_agent_rerankers.NewTextReranker(logger, config, redis),

		// clients
		deploymentClient:  endpoint_client.NewDeploymentServiceClientGRPC(&config.AppConfig, logger, redis),
		vaultClient:       web_client.NewVaultClientGRPC(&config.AppConfig, logger, redis),
		integrationClient: integration_client.NewIntegrationServiceClientGRPC(&config.AppConfig, logger, redis),

		//
		tracer: internal_assistant_telemetry.NewInMemoryTracer(logger,
			internal_assistant_telemetry_exporters.NewLoggingAssistantTraceExporter(logger),
			internal_assistant_telemetry_exporters.NewOpensearchAssistantTraceExporter(
				logger,
				&config.AppConfig, opensearch,
			),
		),

		recorder:          internal_adapter_request_customizers.NewRecorder(logger),
		messaging:         internal_adapter_request_customizers.NewMessaging(logger),
		templateParser:    parsers.NewPongo2StringTemplateParser(logger),
		assistantExecutor: internal_assistant_executors.NewAssistantExecutor(logger),

		// will change

		histories: make([]*types.Message, 0),
		metrics:   make([]*types.Metric, 0),
		metadata:  make(map[string]interface{}),
		args:      make(map[string]interface{}),
		options:   make(map[string]interface{}),
	}
}

// Context implements internal_adapter_requests.Messaging.
func (dm *GenericRequestor) Context() context.Context {
	return dm.ctx
}

// GetSource implements internal_adapter_requests.Messaging.
func (dm *GenericRequestor) Source() utils.RapidaSource {
	return dm.source
}

func (dm *GenericRequestor) Streamer() internal_adapter_request_streamers.Streamer {
	return dm.streamer
}

func (deb *GenericRequestor) OnCreateMessage(
	ctx context.Context,
	messageId string,
	in *types.Message) error {

	start := time.Now()
	deb.histories = append(deb.histories, in)
	//
	_, err := deb.conversationService.CreateConversationMessage(
		ctx,
		deb.Auth(),
		deb.Source(),
		messageId,
		deb.Assistant().Id,
		deb.Assistant().AssistantProviderId,
		deb.Conversation().Id,
		in)

	if err != nil {
		deb.logger.Error("unable to create message for the user")
		return err
	}
	deb.logger.Benchmark("genericRequestor.OnCreateMessage", time.Since(start))
	return nil

}

/*
UpdateMessage updates an existing assistant conversation message in the database.

This function performs the following tasks:
1. Starts a timer to measure the execution time.
2. Initializes span metadata for tracking.
3. Starts an event for message creation using the event manager.
4. Calls the conversation service to update the message in the database.
5. Logs the execution time for benchmarking purposes.
6. Handles any errors that occur during the update process.
7. Updates the span metadata with status information.
8. Returns the updated message or an error.

Parameters:
- message: A pointer to the AssistantConversationMessage to be updated.

Returns:
- A pointer to the updated internal_gorm.AssistantConversationMessage.
- An error if the update operation fails.
*/
func (deb *GenericRequestor) OnUpdateMessage(
	ctx context.Context,
	messageId string,
	message *types.Message,
	status type_enums.RecordState) error {
	start := time.Now()
	spanMetadata := map[string]interface{}{
		"execute": "sequential",
	}
	// appending response
	deb.histories = append(deb.histories, message)

	_, err := deb.conversationService.
		UpdateConversationMessage(
			ctx,
			deb.Auth(),
			deb.Conversation().Id,
			messageId,
			message,
			status,
		)
	deb.logger.Benchmark("GenericRequestor.OnUpdateMessage", time.Since(start))
	if err != nil {
		deb.logger.Errorf("error updating conversation message: %v", err)
		spanMetadata["status"] = "error"
		spanMetadata["error"] = err.Error()
		return err
	}
	return nil
}

func (deb *GenericRequestor) OnMessageMetric(
	ctx context.Context,
	messageId string,
	metrics []*types.Metric) error {
	_, err := deb.
		conversationService.
		ApplyMessageMetrics(
			ctx,
			deb.
				Auth(),
			deb.
				Conversation().Id,
			messageId,
			metrics,
		)

	if err != nil {
		deb.logger.Errorf("error updating metrics for message: %v", err)
		return err
	}
	return nil
}

func (deb *GenericRequestor) OnMessageMetadata(
	ctx context.Context,
	messageId string,
	metadata map[string]interface{}) error {
	start := time.Now()
	_, err := deb.conversationService.
		ApplyMessageMetadata(
			ctx,
			deb.Auth(),
			deb.assistantConversation.Id,
			messageId,
			metadata,
		)

	if err != nil {
		deb.logger.Errorf("error updating metadata for message: %v", err)
		return err
	}
	deb.logger.Benchmark("GenericRequestor.OnMessageMetric", time.Since(start))
	return nil
}

func (dm *GenericRequestor) Tracer() internal_telemetry.VoiceAgentTracer {
	return dm.tracer
}

func (gr *GenericRequestor) GetAssistantConversation(
	auth types.SimplePrinciple,
	assistantId uint64,
	assistantConversationId uint64,
	identifier string,
) (*internal_conversation_gorm.AssistantConversation, error) {
	start := time.Now()
	defer gr.logger.Benchmark("GenericRequestor.GetAssistantConversation", time.Since(start))
	return gr.conversationService.
		GetConversation(
			gr.Context(),
			auth,
			identifier,
			assistantId,
			assistantConversationId,
			&internal_services.
				GetConversationOption{
				InjectContext:  true,
				InjectArgument: true,
				InjectMetadata: true,
				InjectOption:   true,
				InjectMetric:   false},
		)
}

func (gr *GenericRequestor) CreateAssistantConversation(
	auth types.SimplePrinciple,
	assistantId uint64,
	assistantProviderModelId uint64,
	identifier string,
	direction type_enums.ConversationDirection,
	arguments map[string]interface{},
	metadata map[string]interface{},
	options map[string]interface{},
) (*internal_conversation_gorm.AssistantConversation, error) {
	conversation, err := gr.conversationService.CreateConversation(
		gr.Context(),
		auth,
		identifier,
		assistantId,
		assistantProviderModelId,
		direction,
		gr.Source(),
	)
	if err != nil {
		return conversation, err
	}
	utils.Go(gr.Context(), func() {
		gr.CreateConversationArgument(auth, conversation.Id, arguments)
	})

	utils.Go(gr.Context(), func() {
		gr.CreateConversationOption(auth, conversation.Id, options)
	})

	utils.Go(gr.Context(), func() {
		gr.CreateConversationMetadata(auth, conversation.Id, metadata)
	})

	return conversation, err

}

func (gr *GenericRequestor) CreateConversationArgument(auth types.SimplePrinciple, assistantConversationId uint64, args map[string]interface{}) ([]*internal_conversation_gorm.AssistantConversationArgument, error) {
	return gr.conversationService.ApplyConversationArgument(gr.Context(),
		auth,
		assistantConversationId,
		args)
}

func (gr *GenericRequestor) CreateConversationMetadata(auth types.SimplePrinciple, assistantConversationId uint64, metadata map[string]interface{}) ([]*internal_conversation_gorm.AssistantConversationMetadata, error) {
	return gr.conversationService.ApplyConversationMetadata(gr.Context(),
		auth,
		assistantConversationId,
		metadata)
}

func (gr *GenericRequestor) CreateConversationOption(auth types.SimplePrinciple, assistantConversationId uint64, opts map[string]interface{}) ([]*internal_conversation_gorm.AssistantConversationOption, error) {
	return gr.conversationService.ApplyConversationOption(gr.Context(),
		auth,
		assistantConversationId,
		opts)
}

func (talking *GenericRequestor) BeginConversation(
	auth types.SimplePrinciple,
	assistant *internal_assistant_gorm.Assistant,
	direction type_enums.ConversationDirection,
	identifier string,
	argument, metadata, options map[string]interface{}) (*internal_conversation_gorm.AssistantConversation, error) {
	start := time.Now()
	talking.assistant = assistant
	talking.args = argument
	talking.options = options
	talking.metadata = metadata
	conversation, err := talking.
		CreateAssistantConversation(
			auth,
			assistant.Id,
			assistant.AssistantProviderId,
			identifier,
			direction,
			argument,
			metadata,
			options,
		)
	if err != nil {
		talking.logger.Errorf("unable to initialize assistant %+v", err)
		return nil, err
	}
	talking.assistantConversation = conversation
	talking.logger.Benchmark("talking.BeginConversation", time.Since(start))

	return conversation, err
}

func (talking *GenericRequestor) ResumeConversation(
	auth types.SimplePrinciple,
	assistant *internal_assistant_gorm.Assistant,
	conversationId uint64,
	identifier string) (*internal_conversation_gorm.AssistantConversation, error) {
	start := time.Now()
	talking.assistant = assistant

	var conversation *internal_conversation_gorm.AssistantConversation
	conversation, err := talking.GetAssistantConversation(
		auth,
		assistant.Id,
		conversationId,
		identifier,
	)
	if err != nil {
		talking.logger.Errorf("failed to get assistant conversation: %+v", err)
	}

	talking.assistantConversation = conversation
	talking.args = conversation.GetArugments()
	talking.options = conversation.GetOptions()
	talking.metadata = conversation.GetMetadatas()
	talking.logger.Benchmark("talking.ResumeConversation", time.Since(start))

	return conversation, nil
}

func (talking *GenericRequestor) IntegrationCaller() integration_client.IntegrationServiceClient {
	return talking.
		integrationClient

}

func (talking *GenericRequestor) VaultCaller() web_client.VaultClient {
	return talking.
		vaultClient
}

func (talking *GenericRequestor) DeploymentCaller() endpoint_client.DeploymentServiceClient {
	return talking.
		deploymentClient
}

func (talking *GenericRequestor) GetKnowledge(knowledgeId uint64) (*internal_knowledge_gorm.Knowledge, error) {
	return talking.knowledgeService.
		Get(talking.ctx, talking.auth, knowledgeId)
}

func (gr *GenericRequestor) GetArgs() map[string]interface{} {
	return gr.args
}

func (gr *GenericRequestor) GetOptions() map[string]interface{} {
	return gr.options
}

func (dm *GenericRequestor) GetHistories() []*types.Message {
	return dm.histories
}

func (gr *GenericRequestor) CreateConversationRecording(
	body []byte,
) error {
	_, err := gr.conversationService.CreateConversationRecording(
		gr.ctx,
		gr.auth,
		gr.assistantConversation.Id,
		body)
	if err != nil {
		gr.logger.Errorf("unable to create recording for the conversation id %d with error : %v", err)
		return err
	}
	return err
}
