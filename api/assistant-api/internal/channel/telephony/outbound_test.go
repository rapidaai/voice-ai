package channel_telephony

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

type outboundDispatcherTestStore struct {
	callContext *callcontext.CallContext
	lastStatus  callcontext.CallStatusUpdate
	updateCount int
}

type outboundDispatcherConversationServiceStub struct {
	internal_services.AssistantConversationService

	metricAuth           types.SimplePrinciple
	metricAssistantID    uint64
	metricConversationID uint64
	metrics              []*protos.Metric
	metricRecorded       chan struct{}
	metricOnce           sync.Once

	metadataAuth           types.SimplePrinciple
	metadataAssistantID    uint64
	metadataConversationID uint64
	metadata               []*protos.Metadata
	metadataRecorded       chan struct{}
	metadataOnce           sync.Once
}

func (s *outboundDispatcherConversationServiceStub) CreateOrUpdateConversationMetrics(
	_ context.Context,
	auth types.SimplePrinciple,
	assistantID uint64,
	conversationID uint64,
	metrics []*protos.Metric,
) ([]*internal_conversation_entity.AssistantConversationMetric, error) {
	s.metricAuth = auth
	s.metricAssistantID = assistantID
	s.metricConversationID = conversationID
	s.metrics = metrics
	if s.metricRecorded != nil {
		s.metricOnce.Do(func() { close(s.metricRecorded) })
	}
	return nil, nil
}

func (s *outboundDispatcherConversationServiceStub) CreateOrUpdateConversationMetadata(
	_ context.Context,
	auth types.SimplePrinciple,
	assistantID uint64,
	conversationID uint64,
	metadata []*protos.Metadata,
) ([]*internal_conversation_entity.AssistantConversationMetadata, error) {
	s.metadataAuth = auth
	s.metadataAssistantID = assistantID
	s.metadataConversationID = conversationID
	s.metadata = append(s.metadata, metadata...)
	if s.metadataRecorded != nil {
		s.metadataOnce.Do(func() { close(s.metadataRecorded) })
	}
	return nil, nil
}

func (s *outboundDispatcherTestStore) Save(ctx context.Context, cc *callcontext.CallContext) (string, error) {
	s.callContext = cc
	return cc.ContextID, nil
}

func TestInboundDispatcherSaveCallContextUsesDelegatedContext(t *testing.T) {
	organizationID := uint64(7)
	projectID := uint64(8)
	store := &outboundDispatcherTestStore{}
	dispatcher := &InboundDispatcher{store: store}

	_, err := dispatcher.SaveCallContext(
		context.Background(),
		&types.ServiceScope{OrganizationId: &organizationID, ProjectId: &projectID, CurrentToken: "token"},
		&internal_assistant_entity.Assistant{},
		9,
		&internal_type.CallInfo{},
		"sip",
	)
	if err != nil {
		t.Fatalf("SaveCallContext() error = %v", err)
	}
	if store.callContext == nil || store.callContext.OrganizationID != organizationID || store.callContext.ProjectID != projectID {
		t.Fatalf("saved call context = %+v", store.callContext)
	}
}

func (s *outboundDispatcherTestStore) Get(ctx context.Context, contextID string) (*callcontext.CallContext, error) {
	return s.callContext, nil
}

func (s *outboundDispatcherTestStore) GetByChannelUUID(ctx context.Context, provider string, assistantID uint64, channelUUID string) (*callcontext.CallContext, error) {
	return nil, nil
}

func (s *outboundDispatcherTestStore) Claim(ctx context.Context, contextID string) (*callcontext.CallContext, error) {
	s.callContext.Status = callcontext.StatusClaimed
	return s.callContext, nil
}

func (s *outboundDispatcherTestStore) UpdateField(ctx context.Context, contextID, field, value string) error {
	if field == "status" {
		s.callContext.Status = value
	}
	if field == "channel_uuid" {
		s.callContext.ChannelUUID = value
	}
	return nil
}

func (s *outboundDispatcherTestStore) UpdateCallStatus(ctx context.Context, contextID string, status callcontext.CallStatusUpdate) error {
	if status.ExpectedCallStatus != "" && s.callContext.CallStatus != status.ExpectedCallStatus {
		return nil
	}
	s.lastStatus = status
	s.updateCount++
	s.callContext.CallStatus = status.CallStatus
	s.callContext.CallError = status.CallError
	s.callContext.FailureClass = status.FailureClass
	s.callContext.FailureReason = status.FailureReason
	s.callContext.DisconnectReason = status.DisconnectReason
	s.callContext.Retryable = status.Retryable
	s.callContext.ProviderStatusCode = status.ProviderStatusCode
	if status.CallStatus == callcontext.CallStatusFailed || status.CallStatus == callcontext.CallStatusCancelled {
		s.callContext.Status = callcontext.StatusFailed
	}
	return nil
}

func newOutboundDispatcherTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger(
		commons.EnableConsole(true),
		commons.EnableFile(false),
		commons.Level("error"),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

func TestOutboundDispatcher_PersistSetupFailureDoesNotOverwriteTerminalProviderStatus(t *testing.T) {
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID:    "ctx-terminal",
			Provider:     "unknown",
			Status:       callcontext.StatusFailed,
			CallStatus:   callcontext.CallStatusFailed,
			FailureClass: internal_telephony_base.OutboundFailureClassProviderAPI,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:  store,
		logger: newOutboundDispatcherTestLogger(t),
	}

	_, _ = dispatcher.Dispatch(context.Background(), store.callContext.ContextID)

	if store.updateCount != 0 {
		t.Fatalf("expected terminal provider status to be preserved, got %d updates", store.updateCount)
	}
	if store.callContext.FailureClass != internal_telephony_base.OutboundFailureClassProviderAPI {
		t.Fatalf("expected provider failure class preserved, got %q", store.callContext.FailureClass)
	}
}

func TestOutboundDispatcher_PersistSetupFailureDoesNotOverwriteCancelledProviderStatus(t *testing.T) {
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID:        "ctx-cancelled",
			Provider:         "unknown",
			Status:           callcontext.StatusFailed,
			CallStatus:       callcontext.CallStatusCancelled,
			FailureClass:     internal_telephony_base.OutboundFailureClassRequestCancelled,
			DisconnectReason: internal_telephony_base.OutboundDisconnectReasonRequestCancelled,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:  store,
		logger: newOutboundDispatcherTestLogger(t),
	}

	_, _ = dispatcher.Dispatch(context.Background(), store.callContext.ContextID)

	if store.updateCount != 0 {
		t.Fatalf("expected cancelled provider status to be preserved, got %d updates", store.updateCount)
	}
	if store.callContext.CallStatus != callcontext.CallStatusCancelled {
		t.Fatalf("expected cancelled call status preserved, got %q", store.callContext.CallStatus)
	}
	if store.callContext.DisconnectReason != internal_telephony_base.OutboundDisconnectReasonRequestCancelled {
		t.Fatalf("expected cancelled disconnect reason preserved, got %q", store.callContext.DisconnectReason)
	}
}

func TestOutboundDispatcher_PersistSetupFailureUsesProviderNeutralStatus(t *testing.T) {
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID: "ctx-setup",
			Provider:  "unknown",
			Status:    callcontext.StatusPending,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:  store,
		logger: newOutboundDispatcherTestLogger(t),
	}

	_, _ = dispatcher.Dispatch(context.Background(), store.callContext.ContextID)

	if store.callContext.Status != callcontext.StatusFailed {
		t.Fatalf("expected context failed, got %q", store.callContext.Status)
	}
	if store.lastStatus.FailureClass != internal_telephony_base.OutboundFailureClassSetup {
		t.Fatalf("expected setup failure class, got %q", store.lastStatus.FailureClass)
	}
	if store.lastStatus.DisconnectReason != internal_telephony_base.OutboundDisconnectReasonSetupFailed {
		t.Fatalf("expected setup disconnect reason, got %q", store.lastStatus.DisconnectReason)
	}
}

func TestOutboundDispatcher_StatusReporterPersistsFailureDetails(t *testing.T) {
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID: "ctx-provider-failure",
			Status:    callcontext.StatusPending,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:  store,
		logger: newOutboundDispatcherTestLogger(t),
	}
	statusReporter := dispatcher.NewStatusReporter(store.callContext.ContextID)

	statusReporter(internal_type.ProviderCallStatusUpdate{
		ChannelUUID:        "call-486",
		CallStatus:         callcontext.CallStatusFailed,
		ErrorMessage:       "486 Busy Here",
		FailureClass:       "busy",
		FailureReason:      "Busy Here",
		SLIResult:          "client_error",
		SLIReason:          "outbound_busy",
		DisconnectReason:   "outbound_rejected",
		ProviderStatusCode: 486,
	})

	if store.callContext.ChannelUUID != "call-486" {
		t.Fatalf("expected channel UUID persisted, got %q", store.callContext.ChannelUUID)
	}
	if store.callContext.Status != callcontext.StatusFailed {
		t.Fatalf("expected failed context status, got %q", store.callContext.Status)
	}
	if store.callContext.FailureClass != "busy" {
		t.Fatalf("expected busy failure class, got %q", store.callContext.FailureClass)
	}
	if store.callContext.ProviderStatusCode != 486 {
		t.Fatalf("expected provider status 486, got %d", store.callContext.ProviderStatusCode)
	}
}

func TestOutboundDispatcher_StatusReporterRecordsTerminalObservability(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	service := &outboundDispatcherConversationServiceStub{
		metricRecorded:   make(chan struct{}),
		metadataRecorded: make(chan struct{}),
	}
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID:      "ctx-provider-terminal",
			Provider:       SIP.String(),
			Status:         callcontext.StatusPending,
			AssistantID:    33,
			ConversationID: 44,
			OrganizationID: organizationID,
			ProjectID:      projectID,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:               store,
		logger:              newOutboundDispatcherTestLogger(t),
		conversationService: service,
	}
	statusReporter := dispatcher.NewStatusReporter(store.callContext.ContextID)

	statusReporter(internal_type.ProviderCallStatusUpdate{
		ChannelUUID:        "call-486",
		CallStatus:         callcontext.CallStatusFailed,
		ErrorMessage:       "486 Busy Here",
		FailureClass:       "busy",
		FailureReason:      "Busy Here",
		SLIResult:          "client_error",
		SLIReason:          "outbound_busy",
		DisconnectReason:   "outbound_rejected",
		ProviderStatusCode: 486,
	})

	select {
	case <-service.metricRecorded:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for terminal status metric")
	}
	select {
	case <-service.metadataRecorded:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for terminal status metadata")
	}

	if service.metricAuth == nil {
		t.Fatal("expected terminal status metric to be recorded")
	}
	if service.metricAssistantID != 33 || service.metricConversationID != 44 {
		t.Fatalf("unexpected metric scope: assistant=%d conversation=%d", service.metricAssistantID, service.metricConversationID)
	}
	if len(service.metrics) != 2 {
		t.Fatalf("expected call and conversation status metrics, got %+v", service.metrics)
	}
	metricByName := make(map[string]*protos.Metric, len(service.metrics))
	for _, metric := range service.metrics {
		metricByName[metric.Name] = metric
	}
	if metricByName[observability.MetricCallStatus] == nil ||
		metricByName[observability.MetricCallStatus].Value != observability.MetricCallStatusFailed ||
		metricByName[observability.MetricCallStatus].Description != "Busy Here" {
		t.Fatalf("unexpected call status metric: %+v", metricByName[observability.MetricCallStatus])
	}
	if metricByName[type_enums.CONVERSATION_STATUS.String()] == nil ||
		metricByName[type_enums.CONVERSATION_STATUS.String()].Value != observability.MetricCallStatusFailed ||
		metricByName[type_enums.CONVERSATION_STATUS.String()].Description != "Busy Here" {
		t.Fatalf("unexpected conversation status metric: %+v", metricByName[type_enums.CONVERSATION_STATUS.String()])
	}
	if service.metadataAuth == nil {
		t.Fatal("expected terminal status metadata to be recorded")
	}
	metadataByKey := make(map[string]string, len(service.metadata))
	for _, metadata := range service.metadata {
		metadataByKey[metadata.Key] = metadata.Value
	}
	if metadataByKey[observability.MetadataDisconnectReason] != protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String() {
		t.Fatalf("expected disconnect reason metadata, got %q", metadataByKey[observability.MetadataDisconnectReason])
	}
	if metadataByKey[observability.MetadataDisconnectRawReason] != "486 Busy Here" {
		t.Fatalf("expected disconnect raw reason metadata, got %q", metadataByKey[observability.MetadataDisconnectRawReason])
	}
	if metadataByKey[observability.MetadataFailureClass] != "busy" {
		t.Fatalf("expected failure class metadata, got %q", metadataByKey[observability.MetadataFailureClass])
	}
	if metadataByKey[observability.MetadataFailureReason] != "Busy Here" {
		t.Fatalf("expected failure reason metadata, got %q", metadataByKey[observability.MetadataFailureReason])
	}
	if metadataByKey[observability.MetadataSLIResult] != "client_error" {
		t.Fatalf("expected SLI result metadata, got %q", metadataByKey[observability.MetadataSLIResult])
	}
	if metadataByKey[observability.MetadataSLIReason] != "outbound_busy" {
		t.Fatalf("expected SLI reason metadata, got %q", metadataByKey[observability.MetadataSLIReason])
	}
	if metadataByKey[observability.MetadataProviderStatusCode] != "486" {
		t.Fatalf("expected provider status metadata, got %q", metadataByKey[observability.MetadataProviderStatusCode])
	}
	if metadataByKey[observability.MetadataCallError] != "486 Busy Here" {
		t.Fatalf("expected call error metadata, got %q", metadataByKey[observability.MetadataCallError])
	}
}

func TestOutboundDispatcher_StatusReporterRecordsRingingProgressObservability(t *testing.T) {
	service := &outboundDispatcherConversationServiceStub{
		metricRecorded: make(chan struct{}),
	}
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID:      "ctx-provider-ringing",
			Provider:       SIP.String(),
			Status:         callcontext.StatusPending,
			AssistantID:    33,
			ConversationID: 44,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:               store,
		logger:              newOutboundDispatcherTestLogger(t),
		conversationService: service,
	}
	statusReporter := dispatcher.NewStatusReporter(store.callContext.ContextID)

	statusReporter(internal_type.ProviderCallStatusUpdate{
		ChannelUUID:        "call-180",
		CallStatus:         callcontext.CallStatusRinging,
		DisconnectReason:   "outbound_progress_ringing",
		ProviderStatusCode: 180,
	})

	if store.callContext.CallStatus != callcontext.CallStatusRinging {
		t.Fatalf("expected ringing call status, got %q", store.callContext.CallStatus)
	}
	select {
	case <-service.metricRecorded:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ringing metric")
	}
	if service.metricAuth == nil {
		t.Fatal("expected ringing metric to be recorded")
	}
	if service.metricAssistantID != 33 || service.metricConversationID != 44 {
		t.Fatalf("unexpected metric scope: assistant=%d conversation=%d", service.metricAssistantID, service.metricConversationID)
	}
	if len(service.metrics) != 1 {
		t.Fatalf("expected one ringing metric, got %+v", service.metrics)
	}
	if service.metrics[0].Name != observability.MetricCallStatus ||
		service.metrics[0].Value != observability.MetricCallStatusRinging ||
		service.metrics[0].Description != "outbound_progress_ringing" {
		t.Fatalf("unexpected ringing metric: %+v", service.metrics[0])
	}
	if service.metadataAuth != nil || len(service.metadata) != 0 {
		t.Fatalf("expected no ringing metadata, got auth=%v metadata=%+v", service.metadataAuth, service.metadata)
	}
}

func TestOutboundDispatcher_StatusReporterSkipsRingingWhenContextClaimed(t *testing.T) {
	service := &outboundDispatcherConversationServiceStub{}
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID:      "ctx-provider-claimed",
			Provider:       SIP.String(),
			Status:         callcontext.StatusClaimed,
			CallStatus:     callcontext.CallStatusAnswered,
			AssistantID:    33,
			ConversationID: 44,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:               store,
		logger:              newOutboundDispatcherTestLogger(t),
		conversationService: service,
	}
	statusReporter := dispatcher.NewStatusReporter(store.callContext.ContextID)

	statusReporter(internal_type.ProviderCallStatusUpdate{
		ChannelUUID:        "call-180",
		CallStatus:         callcontext.CallStatusRinging,
		DisconnectReason:   "outbound_progress_ringing",
		ProviderStatusCode: 180,
	})

	if store.callContext.CallStatus != callcontext.CallStatusAnswered {
		t.Fatalf("expected claimed context call status to remain answered, got %q", store.callContext.CallStatus)
	}
	if service.metricAuth != nil || len(service.metrics) != 0 {
		t.Fatalf("expected no claimed-context ringing metric, got auth=%v metrics=%+v", service.metricAuth, service.metrics)
	}
	if service.metadataAuth != nil || len(service.metadata) != 0 {
		t.Fatalf("expected no claimed-context ringing metadata, got auth=%v metadata=%+v", service.metadataAuth, service.metadata)
	}
}

func TestOutboundDispatcher_StatusReporterDoesNotRewriteNewCallStatus(t *testing.T) {
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID:  "ctx-new-status",
			Status:     callcontext.StatusPending,
			CallStatus: callcontext.CallStatusRinging,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:  store,
		logger: newOutboundDispatcherTestLogger(t),
	}
	statusReporter := dispatcher.NewStatusReporter(store.callContext.ContextID)

	statusReporter(internal_type.ProviderCallStatusUpdate{
		ChannelUUID: "call-123",
		CallStatus:  callcontext.CallStatusNew,
	})

	if store.callContext.ChannelUUID != "call-123" {
		t.Fatalf("expected channel UUID persisted, got %q", store.callContext.ChannelUUID)
	}
	if store.callContext.CallStatus != callcontext.CallStatusRinging {
		t.Fatalf("expected call status to remain ringing, got %q", store.callContext.CallStatus)
	}
}

func TestOutboundDispatcher_MonitorConnectTimeoutUsesDurableFailureStatus(t *testing.T) {
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID:  "ctx-timeout",
			Provider:   SIP.String(),
			Status:     callcontext.StatusPending,
			CallStatus: callcontext.CallStatusNew,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:                  store,
		logger:                 newOutboundDispatcherTestLogger(t),
		outboundConnectTimeout: 10 * time.Millisecond,
	}

	go dispatcher.monitorCallConnect(context.Background(), store.callContext.ContextID, store.callContext)

	deadline := time.After(250 * time.Millisecond)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatal("expected answer monitor to persist no-answer after timeout")
		case <-ticker.C:
			if store.updateCount == 0 {
				continue
			}
			if store.callContext.Status != callcontext.StatusFailed {
				t.Fatalf("expected context failed, got %q", store.callContext.Status)
			}
			if store.lastStatus.FailureClass != internal_telephony_base.OutboundFailureClassNoAnswer {
				t.Fatalf("expected no_answer failure class, got %q", store.lastStatus.FailureClass)
			}
			if store.lastStatus.DisconnectReason != internal_telephony_base.OutboundDisconnectReasonNoAnswer {
				t.Fatalf("expected connect timeout disconnect reason, got %q", store.lastStatus.DisconnectReason)
			}
			if !store.lastStatus.Retryable {
				t.Fatal("expected connect timeout to be retryable")
			}
			return
		}
	}
}

func TestOutboundDispatcher_MonitorConnectTimeoutSkipsCallbackUpdatedCallStatus(t *testing.T) {
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID:  "ctx-callback-updated",
			Provider:   SIP.String(),
			Status:     callcontext.StatusPending,
			CallStatus: callcontext.CallStatusRinging,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:                  store,
		logger:                 newOutboundDispatcherTestLogger(t),
		outboundConnectTimeout: 10 * time.Millisecond,
	}

	dispatcher.monitorCallConnect(context.Background(), store.callContext.ContextID, store.callContext)

	if store.updateCount != 0 {
		t.Fatalf("expected callback-updated call status to skip watchdog failure, got %d updates", store.updateCount)
	}
	if store.callContext.Status != callcontext.StatusPending {
		t.Fatalf("expected context status to remain pending for media claim, got %q", store.callContext.Status)
	}
}

func TestOutboundDispatcher_MonitorConnectTimeoutSurvivesRequestCancellation(t *testing.T) {
	store := &outboundDispatcherTestStore{
		callContext: &callcontext.CallContext{
			ContextID:  "ctx-monitor",
			Provider:   SIP.String(),
			Status:     callcontext.StatusPending,
			CallStatus: callcontext.CallStatusNew,
		},
	}
	dispatcher := &OutboundDispatcher{
		store:                  store,
		logger:                 newOutboundDispatcherTestLogger(t),
		outboundConnectTimeout: 10 * time.Millisecond,
	}
	requestContext, cancelRequest := context.WithCancel(context.Background())
	callMonitorContext := context.WithoutCancel(requestContext)
	cancelRequest()

	go dispatcher.monitorCallConnect(callMonitorContext, store.callContext.ContextID, store.callContext)

	deadline := time.After(250 * time.Millisecond)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatal("expected answer monitor to persist no-answer after request context cancellation")
		case <-ticker.C:
			if store.updateCount == 0 {
				continue
			}
			if store.callContext.FailureClass != internal_telephony_base.OutboundFailureClassNoAnswer {
				t.Fatalf("expected no_answer failure class, got %q", store.callContext.FailureClass)
			}
			if store.callContext.DisconnectReason != internal_telephony_base.OutboundDisconnectReasonNoAnswer {
				t.Fatalf("expected no_answer disconnect reason, got %q", store.callContext.DisconnectReason)
			}
			return
		}
	}
}

func TestOutboundDispatcher_SIPConnectTimeoutDerivesFromInviteTimeout(t *testing.T) {
	dispatcher := &OutboundDispatcher{
		cfg: &config.AssistantConfig{
			SIPConfig: &config.SIPConfig{InviteTimeout: 30 * time.Second},
		},
		outboundConnectTimeout: defaultOutboundConnectTimeout,
	}

	got := dispatcher.providerOutboundConnectTimeout(SIP.String())

	if got != 45*time.Second {
		t.Fatalf("expected SIP connect timeout 45s, got %s", got)
	}
}
