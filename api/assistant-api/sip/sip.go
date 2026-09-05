// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_sip

import (
	"context"
	"fmt"
	"sync"

	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	sip_middleware "github.com/rapidaai/api/assistant-api/sip/middleware"
	sip_pipeline "github.com/rapidaai/api/assistant-api/sip/pipeline"
	sip_registration "github.com/rapidaai/api/assistant-api/sip/registration"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
)

// SIPEngine manages a multi-tenant SIP server. Config is resolved per-call
// from each assistant's phone deployment and vault credentials.
type SIPEngine struct {
	cfg    *config.AssistantConfig
	logger commons.Logger
	mu     sync.RWMutex
	server *sip_runtime.Server

	ctx    context.Context
	cancel context.CancelFunc

	postgres   connectors.PostgresConnector
	redis      connectors.RedisConnector
	opensearch connectors.OpenSearchConnector
	storage    storages.Storage

	assistantConversationService internal_services.AssistantConversationService
	assistantToolService         internal_services.AssistantToolService
	assistantService             internal_services.AssistantService
	deploymentService            internal_services.AssistantDeploymentService
	configurationService         internal_services.AssistantConfigurationService
	httpLogService               internal_services.AssistantHTTPLogService
	callContextStore             callcontext.Store
	rapidaClient                 *rapida_client.RapidaClient

	// Registration client for maintaining SIP REGISTER with external providers.
	registrationClient *sip_runtime.RegistrationClient

	// Distributed registration manager — runs the GetRecord -> ClaimOwner ->
	// Register -> UpdateStatus pipeline, sharded across instances by externalIP.
	regManager sip_registration.Manager

	// Pipeline dispatcher — routes SIP call lifecycle through extensible stages.
	dispatcher *sip_pipeline.Dispatcher
}

func NewSIPEngine(config *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
	vectordb connectors.VectorConnector,
	rapidaClient *rapida_client.RapidaClient) *SIPEngine {
	fileStorage := storage_files.NewStorage(config.AssetStoreConfig, logger)
	return &SIPEngine{
		cfg:                          config,
		logger:                       logger,
		postgres:                     postgres,
		redis:                        redis,
		opensearch:                   opensearch,
		assistantConversationService: internal_assistant_service.NewAssistantConversationService(logger, postgres, fileStorage),
		assistantToolService:         internal_assistant_service.NewAssistantToolService(logger, postgres, fileStorage),
		assistantService:             internal_assistant_service.NewAssistantService(config, logger, postgres, opensearch),
		deploymentService:            internal_assistant_service.NewAssistantDeploymentService(config, logger, postgres),
		configurationService:         internal_assistant_service.NewAssistantConfigurationService(logger, postgres),
		httpLogService:               internal_assistant_service.NewAssistantHTTPLogService(logger, postgres, fileStorage),
		storage:                      fileStorage,
		callContextStore:             callcontext.NewStore(postgres, logger),
		rapidaClient:                 rapidaClient,
	}
}

func (m *SIPEngine) listenConfig() *sip_runtime.ListenConfig {
	transportType := sip_runtime.TransportUDP
	switch m.cfg.SIPConfig.Transport {
	case "tcp":
		transportType = sip_runtime.TransportTCP
	case "tls":
		transportType = sip_runtime.TransportTLS
	}
	return &sip_runtime.ListenConfig{
		Address:                 m.cfg.SIPConfig.Server,
		ExternalIP:              m.cfg.SIPConfig.ExternalIP,
		AllowLoopbackExternalIP: m.cfg.SIPConfig.AllowLoopbackExternalIP,
		Port:                    m.cfg.SIPConfig.Port,
		Transport:               transportType,
	}
}

// Connect initializes the SIP server. The middleware chain resolves the
// assistant from the SIP route user in the To-URI:
func (m *SIPEngine) Connect(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)
	server, err := sip_runtime.NewServer(m.ctx, &sip_runtime.ServerConfig{
		ListenConfig:         m.listenConfig(),
		Logger:               m.logger,
		RTPPortRangeStart:    m.cfg.SIPConfig.RTPPortRangeStart,
		RTPPortRangeEnd:      m.cfg.SIPConfig.RTPPortRangeEnd,
		SymmetricRTP:         m.cfg.SIPConfig.SymmetricRTP,
		IgnoreLocalAddrInSDP: m.cfg.SIPConfig.IgnoreLocalAddrInSDP,
	})
	if err != nil {
		return fmt.Errorf("failed to create SIP server: %w", err)
	}
	server.SetMiddlewares(
		[]sip_runtime.Middleware{
			sip_middleware.NewRouteMiddleware(
				sip_middleware.WithContext(m.ctx),
				sip_middleware.WithLogger(m.logger),
				sip_middleware.WithAssistantService(m.assistantService),
				sip_middleware.WithServiceID(m.cfg.ServiceID),
			),
			sip_middleware.NewVaultMiddleware(
				sip_middleware.WithContext(m.ctx),
				sip_middleware.WithLogger(m.logger),
				sip_middleware.WithRapidaClient(m.rapidaClient),
				sip_middleware.WithApplySIPConfigDefaults(m.applySIPConfigDefaults),
			),
		},
	)
	server.SetOnApplicationReady(m.onApplicationReady)
	server.SetOnApplicationCleanup(m.onApplicationCleanup)
	server.SetOnInvite(m.onInvite)
	server.SetOnBye(m.onBye)
	server.SetOnCancel(m.onCancel)
	server.SetOnError(m.onError)
	m.registrationClient = sip_runtime.NewRegistrationClient(server.Client(), server.GetListenConfig(), m.logger)
	m.regManager = sip_registration.New(
		sip_registration.WithLogger(m.logger),
		sip_registration.WithPostgres(m.postgres),
		sip_registration.WithRedis(m.redis),
		sip_registration.WithRegistrationClient(m.registrationClient),
		sip_registration.WithAssistantConfig(m.cfg),
		sip_registration.WithSIPConfig(m.cfg.SIPConfig),
		sip_registration.WithApplyOpDefaults(m.applySIPOperationalDefaults),
		sip_registration.WithRapidaClient(m.rapidaClient),
	)

	m.dispatcher = sip_pipeline.New(
		sip_pipeline.WithLogger(m.logger),
		sip_pipeline.WithServer(server),
		sip_pipeline.WithAssistantConfig(m.cfg),
		sip_pipeline.WithAssistantService(m.assistantService),
		sip_pipeline.WithAssistantConversationService(m.assistantConversationService),
		sip_pipeline.WithAssistantToolService(m.assistantToolService),
		sip_pipeline.WithAssistantConfigurationService(m.configurationService),
		sip_pipeline.WithHTTPLogService(m.httpLogService),
		sip_pipeline.WithCallContextStore(m.callContextStore),
		sip_pipeline.WithPostgres(m.postgres),
		sip_pipeline.WithOpenSearch(m.opensearch),
		sip_pipeline.WithRedis(m.redis),
		sip_pipeline.WithStorage(m.storage),
		sip_pipeline.WithRapidaClient(m.rapidaClient),
	)
	m.dispatcher.Start(m.ctx)

	// Start server AFTER dispatcher is ready — incoming INVITEs call m.dispatcher.OnPipeline
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start SIP server: %w", err)
	}
	m.server = server

	// Initial registration sync — runs before returning so DIDs are active before calls arrive.
	m.regManager.Reconcile(m.ctx)

	// Background watcher — polls DB every 5 minutes for new/removed/changed deployments.
	go m.regManager.Start(m.ctx)

	return nil
}

// applySIPOperationalDefaults overlays the engine-level SIP defaults (port,
// transport, RTP range, timeouts, inbound answer policy) onto a per-DID vault
// config. Passed to the registration manager as an injection point so the
// registration package stays decoupled from the assistant-api config types.
func (m *SIPEngine) applySIPOperationalDefaults(c *sip_runtime.Config) {
	if m.cfg == nil || m.cfg.SIPConfig == nil {
		return
	}
	m.applySIPConfigDefaults(c)
}

func (m *SIPEngine) applySIPConfigDefaults(c *sip_runtime.Config) {
	if c == nil || m.cfg == nil || m.cfg.SIPConfig == nil {
		return
	}
	c.ApplyOperationalDefaults(
		m.cfg.SIPConfig.Port,
		sip_runtime.Transport(m.cfg.SIPConfig.Transport),
		m.cfg.SIPConfig.RTPPortRangeStart,
		m.cfg.SIPConfig.RTPPortRangeEnd,
	)
	c.ApplyTimeoutDefaults(
		m.cfg.SIPConfig.RegisterTimeout,
		m.cfg.SIPConfig.InviteTimeout,
		m.cfg.SIPConfig.SessionTimeout,
	)
	c.ApplyMediaTimeoutDefaults(
		m.cfg.SIPConfig.MediaTimeoutInitial,
		m.cfg.SIPConfig.MediaTimeout,
	)
	inboundConfig := m.cfg.SIPConfig.Inbound
	c.ApplyInboundAnswerDefaults(
		sip_runtime.InboundAnswerMode(inboundConfig.AnswerMode),
		inboundConfig.MinRingDuration,
		inboundConfig.MaxRingDuration,
		inboundConfig.ACKTimeout,
	)
}

func (m *SIPEngine) GetServer() *sip_runtime.Server {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.server
}

func (m *SIPEngine) onApplicationReady(session *sip_runtime.Session, requestURI string, callAddress sip_runtime.CallAddress) error {
	stage, err := m.sessionEstablishedStage(session, requestURI, callAddress)
	if err != nil {
		return err
	}
	if m.dispatcher == nil {
		return fmt.Errorf("SIP dispatcher not initialized")
	}
	return m.dispatcher.PrepareSession(m.ctx, stage)
}

func (m *SIPEngine) onApplicationCleanup(session *sip_runtime.Session) {
	if m.dispatcher == nil || session == nil {
		return
	}
	m.dispatcher.DiscardPreparedSession(m.ctx, session.GetCallID())
}

func (m *SIPEngine) onInvite(session *sip_runtime.Session, requestURI string, callAddress sip_runtime.CallAddress) error {
	stage, err := m.sessionEstablishedStage(session, requestURI, callAddress)
	if err != nil {
		return err
	}
	if m.dispatcher == nil {
		return fmt.Errorf("SIP dispatcher not initialized")
	}
	if stage.Direction == sip_runtime.CallDirectionInbound {
		return m.dispatcher.StartPreparedSession(m.ctx, stage)
	}
	m.dispatcher.OnPipeline(m.ctx, stage)
	return nil
}

func (m *SIPEngine) sessionEstablishedStage(session *sip_runtime.Session, requestURI string, callAddress sip_runtime.CallAddress) (sip_pipeline.SessionEstablishedPipeline, error) {
	if session == nil {
		return sip_pipeline.SessionEstablishedPipeline{}, fmt.Errorf("session is nil")
	}
	info := session.GetInfo()
	callID := info.CallID

	if session.IsEnded() {
		return sip_pipeline.SessionEstablishedPipeline{}, fmt.Errorf("session already ended")
	}

	auth := session.GetAuth()
	if auth == nil {
		return sip_pipeline.SessionEstablishedPipeline{}, fmt.Errorf("missing auth on session %s", callID)
	}

	assistant := session.GetAssistant()
	if assistant == nil {
		return sip_pipeline.SessionEstablishedPipeline{}, fmt.Errorf("missing assistant context on session %s", callID)
	}

	return sip_pipeline.SessionEstablishedPipeline{
		ID:              callID,
		Session:         session,
		Config:          session.GetConfig(),
		VaultCredential: session.GetVaultCredential(),
		Direction:       info.Direction,
		AssistantID:     assistant.Id,
		Auth:            auth,
		RequestURI:      requestURI,
		CallAddress:     callAddress,
		ConversationID:  session.GetConversationID(),
	}, nil
}

func (m *SIPEngine) onBye(session *sip_runtime.Session) error {
	disconnectMetadata := session.GetDisconnectMetadata()
	m.persistRemoteByeCallStatus(session, disconnectMetadata)
	return nil
}

func (m *SIPEngine) persistRemoteByeCallStatus(session *sip_runtime.Session, metadata sip_runtime.DisconnectMetadata) {
	if m.callContextStore == nil || session == nil {
		return
	}
	contextID := session.GetContextID()
	if contextID == "" {
		assistant := session.GetAssistant()
		if assistant == nil {
			m.logger.Debugw("SIP BYE status persistence skipped: assistant missing",
				"call_id", session.GetCallID())
			return
		}
		callContext, err := m.callContextStore.GetByChannelUUID(context.Background(), "sip", assistant.Id, session.GetCallID())
		if err != nil {
			m.logger.Debugw("SIP BYE status persistence skipped: call context missing",
				"call_id", session.GetCallID(),
				"assistant_id", assistant.Id,
				"error", err)
			return
		}
		contextID = callContext.ContextID
	}
	if contextID == "" {
		return
	}

	current, err := m.callContextStore.Get(context.Background(), contextID)
	if err == nil && callContextHasTerminalFailure(current) {
		return
	}
	if metadata.Reason == "" {
		metadata.Reason = sip_runtime.DisconnectReasonRemoteHangup
	}
	if err := m.callContextStore.UpdateCallStatus(context.Background(), contextID, callcontext.CallStatusUpdate{
		CallStatus:         callcontext.CallStatusCompleted,
		DisconnectReason:   metadata.Reason,
		ProviderStatusCode: metadata.ProviderStatusCode,
	}); err != nil {
		m.logger.Warnw("SIP BYE status persistence failed",
			"call_id", session.GetCallID(),
			"context_id", contextID,
			"disconnect_reason", metadata.Reason,
			"error", err)
	}
}

func callContextHasTerminalFailure(callContext *callcontext.CallContext) bool {
	if callContext == nil {
		return false
	}
	return callContext.Status == callcontext.StatusFailed ||
		callContext.CallStatus == callcontext.StatusFailed ||
		callContext.CallStatus == "cancelled"
}

func (m *SIPEngine) onCancel(session *sip_runtime.Session) error {
	return nil
}

// onError handles SIP-level errors by emitting a CallFailedPipeline event.
// The pipeline handler (signal.go) creates the observer and persists metrics.
func (m *SIPEngine) onError(session *sip_runtime.Session, callErr error) {
	m.logger.Warnw("SIP error", "call_id", session.GetCallID(), "error", callErr)
	m.dispatcher.OnPipeline(m.ctx, sip_pipeline.CallFailedPipeline{
		ID:      session.GetCallID(),
		Session: session,
		Error:   callErr,
	})
}

func (m *SIPEngine) EndCall(callID string) error {
	m.mu.RLock()
	srv := m.server
	m.mu.RUnlock()
	if srv == nil {
		return fmt.Errorf("SIP server not running")
	}
	session, ok := srv.GetSession(callID)
	if !ok {
		return fmt.Errorf("session %s not found", callID)
	}
	return srv.EndCall(session)
}

func (m *SIPEngine) GetActiveCalls() int {
	m.mu.RLock()
	srv := m.server
	m.mu.RUnlock()
	if srv != nil {
		return srv.SessionCount()
	}
	return 0
}

func (m *SIPEngine) Stop() {
	// Release Redis ownership keys BEFORE UnregisterAll — UnregisterAll drains
	// the active-DID set, after which ReleaseAll would have nothing to walk.
	if m.regManager != nil {
		m.regManager.ReleaseAll(context.Background())
	}
	if m.registrationClient != nil {
		m.registrationClient.UnregisterAll(context.Background())
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Lock()
	srv := m.server
	m.server = nil
	m.mu.Unlock()
	if srv != nil {
		srv.Stop()
	}
	m.logger.Infow("SIP Manager stopped")
}

func (m *SIPEngine) Disconnect(ctx context.Context) error {
	m.Stop()
	return nil
}
