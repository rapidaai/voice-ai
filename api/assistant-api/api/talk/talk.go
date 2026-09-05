// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/api/assistant-api/config"
	internal_adapter "github.com/rapidaai/api/assistant-api/internal/adapters"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_grpc "github.com/rapidaai/api/assistant-api/internal/channel/grpc"
	channel_pipeline "github.com/rapidaai/api/assistant-api/internal/channel/pipeline"
	channel_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony"
	internal_webrtc "github.com/rapidaai/api/assistant-api/internal/channel/webrtc"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/billing"
	observability_collector_conversationmetadata "github.com/rapidaai/api/assistant-api/internal/observability/collectors/conversationmetadata"
	observability_collector_conversationmetric "github.com/rapidaai/api/assistant-api/internal/observability/collectors/conversationmetric"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/telemetry"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

type ConversationApi struct {
	cfg        *config.AssistantConfig
	logger     commons.Logger
	postgres   connectors.PostgresConnector
	redis      connectors.RedisConnector
	opensearch connectors.OpenSearchConnector
	storage    storages.Storage

	callContextStore             callcontext.Store
	outboundDispatcher           *channel_telephony.OutboundDispatcher
	inboundDispatcher            *channel_telephony.InboundDispatcher
	channelPipeline              *channel_pipeline.Dispatcher
	assistantConversationService internal_services.AssistantConversationService
	assistantService             internal_services.AssistantService
	configurationService         internal_services.AssistantConfigurationService
	httpLogService               internal_services.AssistantHTTPLogService
	assistantToolService         internal_services.AssistantToolService
	rapidaClient                 *rapida_client.RapidaClient
}

type ConversationGrpcApi struct {
	ConversationApi
}

func (cApi *ConversationApi) Observability(ctx context.Context, auth *types.Authentication, options ...observability.Option) observability.Recorder {
	configuredCollectors := []observability.Collector{
		observability_collector_conversationmetric.New(observability_collector_conversationmetric.Config{
			Logger:              cApi.logger,
			ConversationService: cApi.assistantConversationService,
		}),
		observability_collector_conversationmetadata.New(observability_collector_conversationmetadata.Config{
			Logger:              cApi.logger,
			ConversationService: cApi.assistantConversationService,
		}),
	}
	configuredCollectors = append(configuredCollectors, telemetry.NewWithAssistantConfig(ctx, cApi.logger, cApi.cfg)...)
	configuredCollectors = append(configuredCollectors, billing.New(billing.Config{
		ProductUsageClient: cApi.rapidaClient.ProductUsage,
	}))
	recorderOptions := []observability.Option{
		observability.WithLogger(cApi.logger),
		observability.WithAuth(auth),
		observability.WithContext(ctx),
		observability.WithCollectors(configuredCollectors...),
	}
	recorderOptions = append(recorderOptions, options...)
	return observability.New(recorderOptions...)
}

// newConversationApiCore builds the shared ConversationApi. All three public
// constructors delegate to this so that deps are created exactly once.
func newConversationApiCore(cfg *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
	sipServer *sip_infra.Server,
	rapidaClient *rapida_client.RapidaClient,
) *ConversationApi {
	store := callcontext.NewStore(postgres, logger)
	assistantService := internal_assistant_service.NewAssistantService(cfg, logger, postgres, opensearch)
	fileStorage := storage_files.NewStorage(cfg.AssetStoreConfig, logger)
	conversationService := internal_assistant_service.NewAssistantConversationService(logger, postgres, fileStorage)
	configurationService := internal_assistant_service.NewAssistantConfigurationService(logger, postgres)
	httpLogService := internal_assistant_service.NewAssistantHTTPLogService(logger, postgres, fileStorage)
	assistantToolService := internal_assistant_service.NewAssistantToolService(logger, postgres, fileStorage)
	inbound := channel_telephony.NewInboundDispatcher(
		channel_telephony.WithConfig(cfg),
		channel_telephony.WithLogger(logger),
		channel_telephony.WithStore(store),
		channel_telephony.WithVaultClient(rapidaClient.Vault),
		channel_telephony.WithAssistantService(assistantService),
		channel_telephony.WithConversationService(conversationService),
		channel_telephony.WithTelephonyOption(channel_telephony.TelephonyOption{SIPServer: sipServer}),
	)
	outbound := channel_telephony.NewOutboundDispatcher(
		channel_telephony.WithOutboundConfig(cfg),
		channel_telephony.WithOutboundLogger(logger),
		channel_telephony.WithOutboundStore(store),
		channel_telephony.WithOutboundVaultClient(rapidaClient.Vault),
		channel_telephony.WithOutboundAssistantService(assistantService),
		channel_telephony.WithOutboundConversationService(conversationService),
		channel_telephony.WithOutboundAssistantConfigurationService(configurationService),
		channel_telephony.WithOutboundHTTPLogService(httpLogService),
		channel_telephony.WithOutboundTelephonyOption(channel_telephony.TelephonyOption{SIPServer: sipServer}),
		channel_telephony.WithOutboundRapidaClient(rapidaClient),
	)
	cApi := &ConversationApi{
		cfg:                          cfg,
		logger:                       logger,
		postgres:                     postgres,
		redis:                        redis,
		opensearch:                   opensearch,
		callContextStore:             store,
		outboundDispatcher:           outbound,
		inboundDispatcher:            inbound,
		assistantConversationService: conversationService,
		assistantService:             assistantService,
		configurationService:         configurationService,
		httpLogService:               httpLogService,
		assistantToolService:         assistantToolService,
		storage:                      fileStorage,
		rapidaClient:                 rapidaClient,
		channelPipeline: channel_pipeline.NewDispatcher(
			channel_pipeline.WithLogger(logger),
			channel_pipeline.WithInboundDispatcher(inbound),
			channel_pipeline.WithOutboundDispatcher(outbound),
			channel_pipeline.WithConversationService(conversationService),
			channel_pipeline.WithAssistantService(assistantService),
			channel_pipeline.WithAssistantConfigurationService(configurationService),
			channel_pipeline.WithHTTPLogService(httpLogService),
			channel_pipeline.WithAssistantToolService(assistantToolService),
		),
	}
	return cApi
}

func NewConversationGRPCApi(config *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
	vectordb connectors.VectorConnector,
	sipServer *sip_infra.Server,
	rapidaClient *rapida_client.RapidaClient,
) assistant_api.TalkServiceServer {
	return &ConversationGrpcApi{*newConversationApiCore(config, logger, postgres, redis, opensearch, sipServer, rapidaClient)}
}

func NewWebRtcApi(config *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
	vectordb connectors.VectorConnector,
	sipServer *sip_infra.Server,
	rapidaClient *rapida_client.RapidaClient,
) assistant_api.WebRTCServer {
	return &ConversationGrpcApi{*newConversationApiCore(config, logger, postgres, redis, opensearch, sipServer, rapidaClient)}
}

// Pipeline returns the channel pipeline dispatcher for use by external engines (e.g. AudioSocket).
func (cApi *ConversationApi) Pipeline() *channel_pipeline.Dispatcher {
	return cApi.channelPipeline
}

func NewConversationApi(config *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
	vectordb connectors.VectorConnector,
	sipServer *sip_infra.Server,
	rapidaClient *rapida_client.RapidaClient,
) *ConversationApi {
	return newConversationApiCore(config, logger, postgres, redis, opensearch, sipServer, rapidaClient)
}

// AssistantTalk handles incoming assistant talk requests.
// It establishes a connection with the client and processes the incoming requests.
//
// Parameters:
// - stream: A server stream for handling bidirectional communication with the client.
//
// Returns:
// - An error if any error occurs during the processing of the request.
func (cApi *ConversationGrpcApi) AssistantTalk(stream assistant_api.TalkService_AssistantTalkServer) error {
	auth, authErr := types.Authorize(stream.Context())
	if authErr != nil {
		return status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	source, ok := utils.GetClientSource(stream.Context())
	if !ok {
		cApi.logger.Errorf("unable to resolve the source from the context")
		return errors.New("illegal source")
	}
	observabilityRecorder := cApi.Observability(
		stream.Context(),
		iAuth,
		observability.WithGracePeriod(),
	)
	defer observabilityRecorder.Close(context.Background())

	streamer, err := internal_grpc.New(
		internal_grpc.WithContext(stream.Context()),
		internal_grpc.WithLogger(cApi.logger),
		internal_grpc.WithServer(stream),
		internal_grpc.WithObserver(observabilityRecorder),
		internal_grpc.WithAuth(iAuth),
		internal_grpc.WithAssistantConfigurationService(cApi.configurationService),
		internal_grpc.WithHTTPLogService(cApi.httpLogService),
		internal_grpc.WithAssistantToolService(cApi.assistantToolService),
	)
	if err != nil {
		cApi.logger.Errorf("failed to create grpc streamer: %v", err)
		return err
	}
	talker, err := internal_adapter.New(
		internal_adapter.WithSource(source),
		internal_adapter.WithContext(stream.Context()),
		internal_adapter.WithConfig(cApi.cfg),
		internal_adapter.WithRapidaClient(cApi.rapidaClient),
		internal_adapter.WithLogger(cApi.logger),
		internal_adapter.WithPostgres(cApi.postgres),
		internal_adapter.WithOpenSearch(cApi.opensearch),
		internal_adapter.WithRedis(cApi.redis),
		internal_adapter.WithStorage(cApi.storage),
		internal_adapter.WithStreamer(streamer),
		internal_adapter.WithObserver(observabilityRecorder),
	)
	if err != nil {
		cApi.logger.Errorf("failed to setup talker: %v", err)
		return err
	}

	return talker.Talk(stream.Context(), iAuth)
}

func (cApi *ConversationGrpcApi) WebTalk(stream assistant_api.WebRTC_WebTalkServer) error {
	auth, authErr := types.Authorize(stream.Context())
	if authErr != nil {
		return status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	source, ok := utils.GetClientSource(stream.Context())
	if !ok {
		cApi.logger.Errorf("unable to resolve the source from the context")
		return errors.New("illegal source")
	}
	observabilityRecorder := cApi.Observability(
		stream.Context(),
		iAuth,
		observability.WithGracePeriod(),
	)
	defer observabilityRecorder.Close(context.Background())

	streamer, err := internal_webrtc.New(
		internal_webrtc.WithContext(stream.Context()),
		internal_webrtc.WithLogger(cApi.logger),
		internal_webrtc.WithServer(stream),
		internal_webrtc.WithServerConfig(cApi.cfg.WebRTCConfig),
		internal_webrtc.WithObserver(observabilityRecorder),
		internal_webrtc.WithAuth(iAuth),
		internal_webrtc.WithAssistantConfigurationService(cApi.configurationService),
		internal_webrtc.WithHTTPLogService(cApi.httpLogService),
		internal_webrtc.WithAssistantToolService(cApi.assistantToolService),
	)
	if err != nil {
		cApi.logger.Errorf("failed to create grpc streamer: %v", err)
		return err
	}
	talker, err := internal_adapter.New(
		internal_adapter.WithSource(source),
		internal_adapter.WithContext(stream.Context()),
		internal_adapter.WithConfig(cApi.cfg),
		internal_adapter.WithRapidaClient(cApi.rapidaClient),
		internal_adapter.WithLogger(cApi.logger),
		internal_adapter.WithPostgres(cApi.postgres),
		internal_adapter.WithOpenSearch(cApi.opensearch),
		internal_adapter.WithRedis(cApi.redis),
		internal_adapter.WithStorage(cApi.storage),
		internal_adapter.WithStreamer(streamer),
		internal_adapter.WithObserver(observabilityRecorder),
	)
	if err != nil {
		cApi.logger.Errorf("failed to setup talker: %v", err)
		return err
	}

	return talker.Talk(stream.Context(), iAuth)
}
