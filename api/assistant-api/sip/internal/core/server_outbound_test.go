package core

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeCall_RegistersSessionBeforeInviteAndUsesSessionCallID(t *testing.T) {
	ua, err := sipgo.NewUA()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ua.Close() })

	client, err := sipgo.NewClient(ua)
	require.NoError(t, err)
	requester := &sessionObservationRequester{delegate: newCancelTrackingRequester()}
	client.TxRequester = requester

	contact := sip.ContactHeader{
		Address: sip.Uri{Scheme: "sip", User: "rapida", Host: "127.0.0.1", Port: 5060},
	}
	dialogClientCache := sipgo.NewDialogClientCache(client, contact)
	server := &Server{
		logger:            bridgeTestLogger(),
		listenConfig:      outboundTestListenConfig(),
		rtpPortRangeStart: 19000,
		rtpPortRangeEnd:   19999,
		dialogClientCache: dialogClientCache,
		sessions:          make(map[string]*Session),
		lifecycles:        make(map[string]*CallLifecycle),
	}
	requester.server = server
	server.state.Store(int32(ServerStateRunning))

	cfg := testOutboundConfig()

	session, err := server.MakeCall(context.Background(), cfg, "+15551234567", "+15557654321", MakeCallOptions{})

	require.NoError(t, err)
	require.NotNil(t, session)
	t.Cleanup(func() { _ = server.CancelCall(session, LifecycleReasonEndCall) })
	assert.NotEmpty(t, session.GetCallID())
	assert.NotEmpty(t, requester.observedCallID())
	assert.Equal(t, session.GetCallID(), requester.observedCallID())
	assert.True(t, requester.observedSessionRegistered())
	assert.True(t, requester.observedLifecycleRegistered())
	registeredSession, ok := server.GetSession(session.GetCallID())
	require.True(t, ok)
	assert.Same(t, session, registeredSession)
}

func TestMakeCall_InviteFailureEndsRegisteredSessionAndReportsFailure(t *testing.T) {
	ua, err := sipgo.NewUA()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ua.Close() })

	client, err := sipgo.NewClient(ua)
	require.NoError(t, err)
	requester := &failingInviteRequester{err: errors.New("invite send failed")}
	client.TxRequester = requester

	contact := sip.ContactHeader{
		Address: sip.Uri{Scheme: "sip", User: "rapida", Host: "127.0.0.1", Port: 5060},
	}
	dialogClientCache := sipgo.NewDialogClientCache(client, contact)
	server := &Server{
		logger:            bridgeTestLogger(),
		listenConfig:      outboundTestListenConfig(),
		rtpPortRangeStart: 19000,
		rtpPortRangeEnd:   19999,
		dialogClientCache: dialogClientCache,
		sessions:          make(map[string]*Session),
		lifecycles:        make(map[string]*CallLifecycle),
	}
	server.state.Store(int32(ServerStateRunning))

	var statusUpdate internal_type.ProviderCallStatusUpdate
	session, err := server.MakeCall(context.Background(), testOutboundConfig(), "+15551234567", "+15557654321", MakeCallOptions{
		CallStatusObserver: func(update internal_type.ProviderCallStatusUpdate) {
			statusUpdate = update
		},
	})

	require.Nil(t, session)
	require.Error(t, err)
	assert.Equal(t, requester.observedCallID(), statusUpdate.ChannelUUID)
	assert.Equal(t, string(OutboundCallStatusFailed), statusUpdate.CallStatus)
	assert.Equal(t, "failed to send INVITE: invite send failed", statusUpdate.ErrorMessage)
	assert.Equal(t, string(OutboundFailureSetup), statusUpdate.FailureClass)
	assert.Equal(t, "failed to send INVITE: invite send failed", statusUpdate.FailureReason)
	assert.Equal(t, string(CallTerminationServerError), statusUpdate.SLIResult)
	assert.Equal(t, "outbound_setup_failed", statusUpdate.SLIReason)
	assert.Equal(t, LifecycleReasonOutboundSetupFailure.String(), statusUpdate.DisconnectReason)
	_, ok := server.GetSession(requester.observedCallID())
	assert.False(t, ok)
}

func TestMakeCall_SessionSurvivesRequestContextAfterInvite(t *testing.T) {
	ua, err := sipgo.NewUA()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ua.Close() })

	client, err := sipgo.NewClient(ua)
	require.NoError(t, err)
	requester := &sessionObservationRequester{delegate: newCancelTrackingRequester()}
	client.TxRequester = requester

	contact := sip.ContactHeader{
		Address: sip.Uri{Scheme: "sip", User: "rapida", Host: "127.0.0.1", Port: 5060},
	}
	dialogClientCache := sipgo.NewDialogClientCache(client, contact)
	server := &Server{
		logger:            bridgeTestLogger(),
		listenConfig:      outboundTestListenConfig(),
		rtpPortRangeStart: 19000,
		rtpPortRangeEnd:   19999,
		dialogClientCache: dialogClientCache,
		sessions:          make(map[string]*Session),
		lifecycles:        make(map[string]*CallLifecycle),
	}
	requester.server = server
	server.state.Store(int32(ServerStateRunning))

	requestContext, cancelRequest := context.WithCancel(context.Background())
	session, err := server.MakeCall(requestContext, testOutboundConfig(), "+15551234567", "+15557654321", MakeCallOptions{})

	require.NoError(t, err)
	require.NotNil(t, session)
	cancelRequest()
	time.Sleep(20 * time.Millisecond)
	t.Cleanup(func() { _ = server.CancelCall(session, LifecycleReasonEndCall) })

	select {
	case <-session.Context().Done():
		t.Fatal("session context should not close when the API request context closes after INVITE setup")
	default:
	}
	assert.Equal(t, int32(0), requester.delegate.cancelRequests.Load())
	assert.NotEqual(t, CallStateCancelled, session.GetInfo().State)
	assert.NotEqual(t, CallStateEnded, session.GetInfo().State)
}

func TestPrepareOutboundCallLeg_AppliesTransferBridgeMetadata(t *testing.T) {
	ua, err := sipgo.NewUA()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ua.Close() })

	client, err := sipgo.NewClient(ua)
	require.NoError(t, err)
	requester := &sessionObservationRequester{delegate: newCancelTrackingRequester()}
	client.TxRequester = requester

	contact := sip.ContactHeader{
		Address: sip.Uri{Scheme: "sip", User: "rapida", Host: "127.0.0.1", Port: 5060},
	}
	dialogClientCache := sipgo.NewDialogClientCache(client, contact)
	server := &Server{
		logger:            bridgeTestLogger(),
		listenConfig:      outboundTestListenConfig(),
		rtpPortRangeStart: 19000,
		rtpPortRangeEnd:   19999,
		dialogClientCache: dialogClientCache,
		sessions:          make(map[string]*Session),
		lifecycles:        make(map[string]*CallLifecycle),
	}
	requester.server = server
	server.state.Store(int32(ServerStateRunning))

	outboundCall, err := server.prepareOutboundCallLeg(context.Background(), testOutboundConfig(), "+15551234567", "+15557654321", outboundCallLegOptions{
		purpose:         OutboundLegPurposeTransferBridge,
		makeCallOptions: MakeCallOptions{ContextID: "ctx-transfer"},
		parentCallID:    "parent-call-id",
		parentContextID: "parent-context-id",
		parentConvID:    4001,
		transferTarget:  "+15551234567",
		transferAttempt: 2,
		transferTotal:   3,
	})

	require.NoError(t, err)
	require.NotNil(t, outboundCall)
	t.Cleanup(func() { _ = server.CancelCall(outboundCall.session, LifecycleReasonEndCall) })
	assert.True(t, requester.observedSessionRegistered())
	assert.True(t, requester.observedLifecycleRegistered())
	assertSessionMetadata(t, outboundCall.session, MetadataOutboundLegPurpose, string(OutboundLegPurposeTransferBridge))
	assertSessionMetadata(t, outboundCall.session, MetadataOutboundParentCallID, "parent-call-id")
	assertSessionMetadata(t, outboundCall.session, MetadataOutboundParentContextID, "parent-context-id")
	assertSessionMetadata(t, outboundCall.session, MetadataOutboundParentConversationID, uint64(4001))
	assertSessionMetadata(t, outboundCall.session, MetadataOutboundTransferTarget, "+15551234567")
	assertSessionMetadata(t, outboundCall.session, MetadataOutboundTransferAttempt, 2)
	assertSessionMetadata(t, outboundCall.session, MetadataOutboundTransferTotal, 3)
}

func outboundTestListenConfig() *ListenConfig {
	return &ListenConfig{
		Address:                 "127.0.0.1",
		ExternalIP:              "127.0.0.1",
		AllowLoopbackExternalIP: true,
		Port:                    5060,
		Transport:               TransportUDP,
	}
}

func assertSessionMetadata(t *testing.T, session *Session, key string, expected interface{}) {
	t.Helper()
	value, ok := session.GetMetadata(key)
	require.True(t, ok, "metadata %s missing", key)
	assert.Equal(t, expected, value)
}

type sessionObservationRequester struct {
	mu                sync.Mutex
	server            *Server
	delegate          *cancelTrackingRequester
	callID            string
	sessionRegistered bool
	lifecycleExists   bool
}

func (r *sessionObservationRequester) Request(ctx context.Context, req *sip.Request) (sip.ClientTransaction, error) {
	if req.Method == sip.INVITE {
		r.recordInviteOwnership(req)
	}
	return r.delegate.Request(ctx, req)
}

func (r *sessionObservationRequester) recordInviteOwnership(req *sip.Request) {
	callID := ""
	if header := req.CallID(); header != nil {
		callID = header.Value()
	}

	r.server.mu.RLock()
	_, sessionRegistered := r.server.sessions[callID]
	_, lifecycleExists := r.server.lifecycles[callID]
	r.server.mu.RUnlock()

	r.mu.Lock()
	r.callID = callID
	r.sessionRegistered = sessionRegistered
	r.lifecycleExists = lifecycleExists
	r.mu.Unlock()
}

func (r *sessionObservationRequester) observedCallID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.callID
}

func (r *sessionObservationRequester) observedSessionRegistered() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessionRegistered
}

func (r *sessionObservationRequester) observedLifecycleRegistered() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lifecycleExists
}

type failingInviteRequester struct {
	mu     sync.Mutex
	callID string
	err    error
}

func (r *failingInviteRequester) Request(ctx context.Context, req *sip.Request) (sip.ClientTransaction, error) {
	if req.Method == sip.INVITE {
		r.mu.Lock()
		if header := req.CallID(); header != nil {
			r.callID = header.Value()
		}
		r.mu.Unlock()
		return nil, r.err
	}
	return newFakeClientTransaction(), nil
}

func (r *failingInviteRequester) observedCallID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.callID
}
