// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
)

const (
	signalChSize = 64
	setupChSize  = 256
	mediaChSize  = 256
)

type callEnvelope struct {
	ctx context.Context
	p   sip_infra.Pipeline
}

// Dispatcher routes SIP pipeline stages to priority-based channel goroutines.
// Stateless — no per-call state stored on the Dispatcher.
type Dispatcher struct {
	logger commons.Logger

	signalCh chan callEnvelope
	setupCh  chan callEnvelope
	mediaCh  chan callEnvelope

	preparedMu       sync.Mutex
	preparedSessions map[string]*preparedSession

	server TransferServer

	assistantConfig              *config.AssistantConfig
	assistantService             internal_services.AssistantService
	assistantConversationService internal_services.AssistantConversationService
	assistantToolService         internal_services.AssistantToolService
	configurationService         internal_services.AssistantConfigurationService
	httpLogService               internal_services.AssistantHTTPLogService
	callContextStore             callcontext.Store
	postgres                     connectors.PostgresConnector
	opensearch                   connectors.OpenSearchConnector
	redis                        connectors.RedisConnector
	storage                      storages.Storage
}

type CallSetupResult struct {
	AssistantID         uint64
	ConversationID      uint64
	AssistantProviderId uint64
	Auth                *types.Authentication
	ProjectID           uint64
	OrganizationID      uint64
	CallContext         *callcontext.CallContext
}

type PreparedCallRuntime interface {
	Start(ctx context.Context) error
	Close(ctx context.Context)
}

type DispatcherOptions struct {
	Logger                       commons.Logger
	Server                       *sip_infra.Server
	TransferServer               TransferServer
	AssistantConfig              *config.AssistantConfig
	AssistantService             internal_services.AssistantService
	AssistantConversationService internal_services.AssistantConversationService
	AssistantToolService         internal_services.AssistantToolService
	ConfigurationService         internal_services.AssistantConfigurationService
	HTTPLogService               internal_services.AssistantHTTPLogService
	CallContextStore             callcontext.Store
	Postgres                     connectors.PostgresConnector
	OpenSearch                   connectors.OpenSearchConnector
	Redis                        connectors.RedisConnector
	Storage                      storages.Storage
}

type DispatcherOption func(*DispatcherOptions)

func WithLogger(logger commons.Logger) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.Logger = logger
	}
}

func WithServer(server *sip_infra.Server) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.Server = server
	}
}

func WithTransferServer(server TransferServer) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.TransferServer = server
	}
}

func WithAssistantConfig(assistantConfig *config.AssistantConfig) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.AssistantConfig = assistantConfig
	}
}

func WithAssistantService(assistantService internal_services.AssistantService) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.AssistantService = assistantService
	}
}

func WithAssistantConversationService(assistantConversationService internal_services.AssistantConversationService) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.AssistantConversationService = assistantConversationService
	}
}

func WithAssistantToolService(assistantToolService internal_services.AssistantToolService) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.AssistantToolService = assistantToolService
	}
}

func WithAssistantConfigurationService(configurationService internal_services.AssistantConfigurationService) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.ConfigurationService = configurationService
	}
}

func WithHTTPLogService(httpLogService internal_services.AssistantHTTPLogService) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.HTTPLogService = httpLogService
	}
}

func WithCallContextStore(callContextStore callcontext.Store) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.CallContextStore = callContextStore
	}
}

func WithPostgres(postgres connectors.PostgresConnector) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.Postgres = postgres
	}
}

func WithOpenSearch(opensearch connectors.OpenSearchConnector) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.OpenSearch = opensearch
	}
}

func WithRedis(redis connectors.RedisConnector) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.Redis = redis
	}
}

func WithStorage(storage storages.Storage) DispatcherOption {
	return func(options *DispatcherOptions) {
		options.Storage = storage
	}
}

// TransferServer is the minimal SIP infra surface required by transfer orchestration.
// It enables deterministic tests by allowing fake implementations.
type TransferServer interface {
	sip_infra.LifecycleController
	MakeTransferBridgeCall(ctx context.Context, cfg *sip_infra.Config, toURI, fromURI string, opts sip_infra.TransferBridgeCallOptions) (*sip_infra.Session, error)
	BridgeTransfer(ctx context.Context, inbound, outbound *sip_infra.Session, onOperatorAudio func([]byte)) (sip_infra.BridgeEndReason, error)
}

func New(opts ...DispatcherOption) *Dispatcher {
	options := &DispatcherOptions{}
	for _, opt := range opts {
		opt(options)
	}

	transferServer := options.TransferServer
	if transferServer == nil && options.Server != nil {
		transferServer = options.Server
	}
	return &Dispatcher{
		logger:                       options.Logger,
		server:                       transferServer,
		assistantConfig:              options.AssistantConfig,
		assistantService:             options.AssistantService,
		assistantConversationService: options.AssistantConversationService,
		assistantToolService:         options.AssistantToolService,
		configurationService:         options.ConfigurationService,
		httpLogService:               options.HTTPLogService,
		callContextStore:             options.CallContextStore,
		postgres:                     options.Postgres,
		opensearch:                   options.OpenSearch,
		redis:                        options.Redis,
		storage:                      options.Storage,
		signalCh:                     make(chan callEnvelope, signalChSize),
		setupCh:                      make(chan callEnvelope, setupChSize),
		mediaCh:                      make(chan callEnvelope, mediaChSize),
		preparedSessions:             make(map[string]*preparedSession),
	}
}

func (d *Dispatcher) transitionCall(session *sip_infra.Session, next sip_infra.CallState, reason sip_infra.LifecycleReason) bool {
	if d.server == nil {
		d.logger.Warnw("SIP lifecycle transition skipped: server unavailable",
			"call_id", session.GetCallID(),
			"to", next,
			"reason", reason)
		return false
	}
	return d.server.TransitionCall(session, next, reason)
}

func (d *Dispatcher) endCall(session *sip_infra.Session, reason sip_infra.LifecycleReason) {
	if d.server == nil {
		d.logger.Warnw("SIP lifecycle end skipped: server unavailable",
			"call_id", session.GetCallID(),
			"reason", reason)
		return
	}
	if err := d.server.EndCallWithReason(session, reason); err != nil {
		d.logger.Warnw("SIP lifecycle end failed",
			"call_id", session.GetCallID(),
			"reason", reason,
			"error", err)
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	go d.runDispatcher(ctx, d.signalCh)
	go d.runDispatcher(ctx, d.setupCh)
	go d.runDispatcher(ctx, d.mediaCh)
	d.logger.Infow("SIP pipeline dispatcher started")
}

func (d *Dispatcher) OnPipeline(ctx context.Context, stages ...sip_infra.Pipeline) {
	for _, s := range stages {
		e := callEnvelope{ctx: ctx, p: s}
		switch s.(type) {
		case sip_infra.TransferInitiatedPipeline,
			sip_infra.TransferConnectedPipeline,
			sip_infra.TransferFailedPipeline,
			sip_infra.CallFailedPipeline:
			d.signalCh <- e
		case sip_infra.SessionEstablishedPipeline:
			d.mediaCh <- e
		default:
			d.logger.Warnw("OnPipeline: unrouted type", "type", fmt.Sprintf("%T", s))
		}
	}
}

func (d *Dispatcher) runDispatcher(ctx context.Context, ch chan callEnvelope) {
	for {
		select {
		case <-ctx.Done():
			d.drain(ch)
			return
		case e := <-ch:
			d.dispatch(e.ctx, e.p)
		}
	}
}

func (d *Dispatcher) drain(ch chan callEnvelope) {
	for {
		select {
		case e := <-ch:
			d.dispatch(e.ctx, e.p)
		default:
			return
		}
	}
}

func (d *Dispatcher) dispatch(ctx context.Context, p sip_infra.Pipeline) {
	switch v := p.(type) {
	case sip_infra.SessionEstablishedPipeline:
		d.handleSessionEstablished(ctx, v)
	case sip_infra.TransferInitiatedPipeline:
		d.handleTransferInitiated(ctx, v)
	case sip_infra.TransferConnectedPipeline:
		d.handleTransferConnected(ctx, v)
	case sip_infra.TransferFailedPipeline:
		d.handleTransferFailed(ctx, v)
	case sip_infra.CallFailedPipeline:
		d.handleCallFailed(ctx, v)
	default:
		d.logger.Warnw("dispatch: unknown pipeline type", "type", fmt.Sprintf("%T", p))
	}
}
