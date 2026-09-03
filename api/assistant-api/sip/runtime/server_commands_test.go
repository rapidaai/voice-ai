// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_inbound "github.com/rapidaai/api/assistant-api/sip/internal/inbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testServerTx struct {
	mu        sync.Mutex
	done      chan struct{}
	err       error
	responses []*sip.Response
}

func newTestServerTx() *testServerTx {
	return &testServerTx{
		done:      make(chan struct{}),
		responses: make([]*sip.Response, 0, 2),
	}
}

func (t *testServerTx) Terminate() { close(t.done) }
func (t *testServerTx) OnTerminate(f sip.FnTxTerminate) bool {
	if f != nil {
		f("test-tx", t.err)
	}
	return false
}
func (t *testServerTx) Done() <-chan struct{} { return t.done }
func (t *testServerTx) Err() error            { return t.err }
func (t *testServerTx) Respond(res *sip.Response) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.responses = append(t.responses, res)
	return nil
}
func (t *testServerTx) Acks() <-chan *sip.Request { return nil }
func (t *testServerTx) OnCancel(_ sip.FnTxCancel) bool {
	return false
}

type ackableTestServerTx struct {
	*testServerTx
	acks chan *sip.Request
}

func newAckableTestServerTx() *ackableTestServerTx {
	return &ackableTestServerTx{
		testServerTx: newTestServerTx(),
		acks:         make(chan *sip.Request, 2),
	}
}

func (t *ackableTestServerTx) Acks() <-chan *sip.Request {
	return t.acks
}

func (t *ackableTestServerTx) PushACK(req *sip.Request) {
	t.acks <- req
}

type failingStatusServerTx struct {
	*activeTestServerTx
	statusCode int
}

func newFailingStatusServerTx(statusCode int) *failingStatusServerTx {
	return &failingStatusServerTx{
		activeTestServerTx: newActiveTestServerTx(),
		statusCode:         statusCode,
	}
}

func (t *failingStatusServerTx) Respond(response *sip.Response) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.responses = append(t.responses, response)
	if response.StatusCode == t.statusCode {
		return assert.AnError
	}
	return nil
}

func (t *testServerTx) lastStatus() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.responses) == 0 {
		return 0
	}
	return t.responses[len(t.responses)-1].StatusCode
}

func (t *testServerTx) statusCount(statusCode int) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	count := 0
	for _, response := range t.responses {
		if response.StatusCode == statusCode {
			count++
		}
	}
	return count
}

func newServerForCommandTests(t *testing.T) *Server {
	t.Helper()

	ua, err := sipgo.NewUA()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ua.Close() })

	client, err := sipgo.NewClient(ua)
	require.NoError(t, err)

	contact := sip.ContactHeader{
		Address: sip.Uri{Scheme: "sip", Host: "127.0.0.1", Port: 5060},
	}

	server := &Server{
		logger:            bridgeTestLogger(),
		listenConfig:      &ListenConfig{Address: "127.0.0.1", Port: 5060, ExternalIP: "127.0.0.1"},
		rtpPortRangeStart: 19000,
		rtpPortRangeEnd:   19999,
		dialogClientCache: sipgo.NewDialogClientCache(client, contact),
		dialogServerCache: sipgo.NewDialogServerCache(client, contact),
		sessions:          make(map[string]*Session),
		lifecycles:        make(map[string]*CallLifecycle),
		pendingInvites:    make(map[inboundInviteKey]*pendingInvite),
		cancelledInvites:  make(map[inboundInviteKey]bool),
		inboundACKTimeout: defaultInboundACKTimeout,
		ctx:               context.Background(),
	}
	t.Cleanup(func() {
		server.mu.RLock()
		sessions := make([]*Session, 0, len(server.sessions))
		for _, session := range server.sessions {
			sessions = append(sessions, session)
		}
		server.mu.RUnlock()
		for _, session := range sessions {
			_ = server.EndCallWithReason(session, LifecycleReasonEndCall)
		}
	})
	return server
}

func newSIPRequest(method sip.RequestMethod, callID string) *sip.Request {
	recipient := sip.Uri{Scheme: "sip", User: "bob", Host: "example.com", Port: 5060}
	req := sip.NewRequest(method, recipient)

	params := sip.NewParams()
	params["branch"] = sip.GenerateBranch()
	fromParams := sip.NewParams()
	fromParams.Add("tag", "fromtag")
	req.AppendHeader(&sip.ViaHeader{
		ProtocolName:    "SIP",
		ProtocolVersion: "2.0",
		Transport:       "UDP",
		Host:            "127.0.0.1",
		Port:            5060,
		Params:          params,
	})
	req.AppendHeader(&sip.FromHeader{
		DisplayName: "Alice",
		Address: sip.Uri{
			Scheme: "sip",
			User:   "alice",
			Host:   "example.com",
			Port:   5060,
		},
		Params: fromParams,
	})
	req.AppendHeader(&sip.ToHeader{
		DisplayName: "Bob",
		Address:     recipient,
		Params:      sip.NewParams(),
	})
	ch := sip.CallIDHeader(callID)
	req.AppendHeader(&ch)
	req.AppendHeader(&sip.CSeqHeader{SeqNo: 1, MethodName: method})
	req.SetBody(nil)
	return req
}

func newInboundInviteRequest(callID string) *sip.Request {
	req := newSIPRequest(sip.INVITE, callID)
	req.AppendHeader(&sip.ContactHeader{
		Address: sip.Uri{Scheme: "sip", User: "alice", Host: "127.0.0.1", Port: 5060},
	})
	req.AppendHeader(sip.NewHeader("Content-Type", internal_inbound.SDPContentType))
	req.SetBody([]byte(validInboundOfferSDP()))
	return req
}

func newDialogSDPRequest(method sip.RequestMethod, callID string, sdpBody string) *sip.Request {
	req := newSIPRequest(method, callID)
	if fromHeader := req.From(); fromHeader != nil {
		fromHeader.Params.Add("tag", "fromtag")
	}
	req.AppendHeader(sip.NewHeader("Content-Type", internal_inbound.SDPContentType))
	req.SetBody([]byte(sdpBody))
	return req
}

func newACKRequest(callID string) *sip.Request {
	return newSIPRequest(sip.ACK, callID)
}

func validInboundOfferSDP() string {
	return "v=0\r\n" +
		"o=caller 1 1 IN IP4 127.0.0.1\r\n" +
		"s=call\r\n" +
		"c=IN IP4 127.0.0.1\r\n" +
		"t=0 0\r\n" +
		"m=audio 19000 RTP/AVP 0 101\r\n" +
		"a=rtpmap:0 PCMU/8000\r\n" +
		"a=rtpmap:101 telephone-event/8000\r\n" +
		"a=sendrecv\r\n"
}

func newTestSession(t *testing.T, callID string, dir CallDirection) *Session {
	t.Helper()
	s, err := NewSession(context.Background(),
		WithSessionConfig(bridgeTestConfig()),
		WithSessionDirection(dir),
		WithSessionCallID(callID),
		WithSessionCodec(&CodecPCMU),
	)
	require.NoError(t, err)
	return s
}

func registerConnectedInboundSession(t *testing.T, s *Server, callID string) *Session {
	t.Helper()

	session := newTestSession(t, callID, CallDirectionInbound)
	session.SetLocalRTP("127.0.0.1", 18000)
	session.SetRemoteRTP("127.0.0.1", 19000)
	s.registerSession(session, callID)
	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonInboundInviteRinging))
	require.True(t, s.TransitionCall(session, CallStateConnected, LifecycleReasonInboundInviteAnswered))
	require.True(t, session.MarkInitialACKReceived())
	return session
}

func registerConnectedInboundDialogSession(t *testing.T, s *Server, callID string) *Session {
	t.Helper()

	request := newInboundInviteRequest(callID)
	transaction := newActiveTestServerTx()
	inboundCall := NewInbound(s, request, transaction)
	loadInboundIdentity(t, inboundCall)
	loadInboundMediaOffer(t, inboundCall)
	inboundCall.resolvedConfig = inboundConfig{config: bridgeTestConfig()}
	createInboundSessionForTest(t, inboundCall)
	inboundCall.session.SetRemoteRTP("127.0.0.1", 19000)
	s.registerSession(inboundCall.session, inboundCall.identity.callID)
	createInboundDialogForTest(t, inboundCall)
	require.True(t, s.TransitionCall(inboundCall.session, CallStateRinging, LifecycleReasonInboundInviteRinging))
	require.True(t, s.TransitionCall(inboundCall.session, CallStateConnected, LifecycleReasonInboundInviteAnswered))
	require.True(t, inboundCall.session.MarkInitialACKReceived())
	return inboundCall.session
}

func newInboundDialogRequest(t *testing.T, session *Session, method sip.RequestMethod) *sip.Request {
	t.Helper()

	dialogSession := session.GetDialogServerSession()
	require.NotNil(t, dialogSession)
	require.NotNil(t, dialogSession.InviteRequest)

	request := newSIPRequest(method, session.GetCallID())
	if fromTag, ok := dialogSession.InviteRequest.From().Params.Get("tag"); ok {
		request.From().Params.Add("tag", fromTag)
	}
	if toTag, ok := dialogSession.InviteRequest.To().Params.Get("tag"); ok {
		request.To().Params.Add("tag", toTag)
	}
	if cseq := request.CSeq(); cseq != nil {
		cseq.SeqNo = 2
	}
	return request
}

func newInboundDialogSDPRequest(t *testing.T, session *Session, method sip.RequestMethod, sdpBody string) *sip.Request {
	t.Helper()
	request := newInboundDialogRequest(t, session, method)
	request.AppendHeader(sip.NewHeader("Content-Type", internal_inbound.SDPContentType))
	request.SetBody([]byte(sdpBody))
	return request
}

func TestSIPCommand_INVITE_NoResolver_Rejects500(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newInboundInviteRequest("call-invite-1")
	tx := newTestServerTx()

	s.handleInvite(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 500, tx.lastStatus())
}

func TestSIPCommand_ACK_InboundSession_MarksConnected(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.ACK, "call-ack-1")
	tx := newTestServerTx()

	session := newTestSession(t, "call-ack-1", CallDirectionInbound)
	session.SetState(CallStateRinging)
	s.sessions["call-ack-1"] = session

	s.handleAck(req, tx)

	assert.Equal(t, CallStateConnected, session.GetState())
	assert.True(t, session.HasInitialACKReceived())
}

func TestSIPCommand_ACK_ReInviteACKTrackedSeparately(t *testing.T) {
	s := newServerForCommandTests(t)
	session := registerConnectedInboundSession(t, s, "call-ack-reinvite")
	session.BeginReInviteACKWait()
	req := newSIPRequest(sip.ACK, "call-ack-reinvite")
	tx := newTestServerTx()

	s.handleAck(req, tx)

	assert.Equal(t, CallStateConnected, session.GetState())
	assert.Equal(t, uint64(1), session.ReInviteACKCount())
}

func TestSIPCommand_ACK_LateACKDoesNotMutateLifecycle(t *testing.T) {
	s := newServerForCommandTests(t)
	session := registerConnectedInboundSession(t, s, "call-ack-late")
	req := newSIPRequest(sip.ACK, "call-ack-late")
	tx := newTestServerTx()

	s.handleAck(req, tx)

	assert.Equal(t, CallStateConnected, session.GetState())
	assert.Equal(t, uint64(0), session.ReInviteACKCount())
}

func TestSIPCommand_InitialAnswerRequiresDialogOwnership(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-ack-timeout", CallDirectionInbound)
	req := newInboundInviteRequest("call-ack-timeout")
	tx := newAckableTestServerTx()

	err := s.sendSDPResponseAndWaitACK(tx, req, session, validInboundOfferSDP(), LifecycleReasonInboundInviteACKReceived, s.effectiveInboundACKTimeout(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires dialog ownership")
	assert.Empty(t, tx.responses)
	assert.False(t, session.HasInitialACKReceived())
}

func TestSIPCommand_InitialAnswerWakesWhenACKHandledOutsideTransaction(t *testing.T) {
	s := newServerForCommandTests(t)
	s.inboundACKTimeout = 500 * time.Millisecond
	request := newInboundInviteRequest("call-ack-outside-tx")
	inviteTransaction := newActiveTestServerTx()
	inboundCall := NewInbound(s, request, inviteTransaction)
	loadInboundIdentity(t, inboundCall)
	loadInboundMediaOffer(t, inboundCall)
	inboundCall.resolvedConfig = inboundConfig{config: bridgeTestConfig()}
	createInboundSessionForTest(t, inboundCall)
	s.registerSession(inboundCall.session, inboundCall.identity.callID)
	createInboundDialogForTest(t, inboundCall)
	require.True(t, s.TransitionCall(inboundCall.session, CallStateRinging, LifecycleReasonInboundInviteRinging))

	answerTransaction := newAckableTestServerTx()
	answerDone := make(chan error, 1)
	go func() {
		answerDone <- s.sendSDPResponseAndWaitACK(
			answerTransaction,
			inboundCall.session.GetDialogServerSession().InviteRequest,
			inboundCall.session,
			validInboundOfferSDP(),
			LifecycleReasonInboundInviteACKReceived,
			s.effectiveInboundACKTimeout(),
			nil,
		)
	}()

	require.Eventually(t, func() bool {
		return answerTransaction.lastStatus() == 200
	}, time.Second, time.Millisecond)

	ackStartedAt := time.Now()
	s.handleAck(newInboundDialogRequest(t, inboundCall.session, sip.ACK), newTestServerTx())

	select {
	case err := <-answerDone:
		require.NoError(t, err)
		assert.Less(t, time.Since(ackStartedAt), 250*time.Millisecond)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for external ACK to release initial answer wait")
	}
	require.NotEmpty(t, answerTransaction.responses)
	assert.Equal(t, 200, answerTransaction.lastStatus())
	assert.True(t, inboundCall.session.HasInitialACKReceived())
}

func TestSIPCommand_BYE_InboundSession_NotifiesAndEnds(t *testing.T) {
	s := newServerForCommandTests(t)
	session := registerConnectedInboundDialogSession(t, s, "call-bye-1")
	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})
	onByeCalled := false
	s.SetOnBye(func(byeSession *Session) error {
		onByeCalled = true
		assert.True(t, byeSession.IsEnded(), "inbound remote BYE should end before onBye callback")
		return s.EndCallWithReason(byeSession, LifecycleReasonRemoteBye)
	})
	req := newInboundDialogRequest(t, session, sip.BYE)
	req.AppendHeader(sip.NewHeader("Reason", `Q.850;cause=16;text="Normal call clearing"`))
	tx := newTestServerTx()

	s.handleBye(req, tx)

	assert.True(t, session.IsEnded())
	select {
	case <-session.ByeReceived():
	default:
		t.Fatalf("expected ByeReceived signal")
	}
	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 200, tx.lastStatus())
	assert.True(t, onByeCalled)
	metadata := session.GetDisconnectMetadata()
	assert.Equal(t, DisconnectReasonNormalClearing, metadata.Reason)
	assert.Equal(t, 16, metadata.ProviderStatusCode)
	assert.Equal(t, "Normal call clearing", metadata.Text)
	assert.False(t, disconnectCalled, "remote BYE must not trigger local BYE")

	metadataValue, ok := session.GetMetadata(MetadataDisconnectReason)
	require.True(t, ok)
	assert.Equal(t, DisconnectReasonNormalClearing, metadataValue)
	metadataValue, ok = session.GetMetadata(MetadataDisconnectRawReason)
	require.True(t, ok)
	assert.Equal(t, `Q.850;cause=16;text="Normal call clearing"`, metadataValue)
}

func TestSIPCommand_BYE_OutboundSession_NotifiesAndEndsWithoutLocalBYE(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-bye-outbound", CallDirectionOutbound)
	session.SetState(CallStateConnected)
	session.SetOutboundDialogPhase(OutboundDialogPhaseConfirmed)
	s.registerSession(session, "call-bye-outbound")

	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})
	onByeCalled := false
	s.SetOnBye(func(byeSession *Session) error {
		onByeCalled = true
		assert.True(t, byeSession.IsEnded(), "outbound remote BYE should end before onBye callback")
		return s.EndCallWithReason(byeSession, LifecycleReasonRemoteBye)
	})

	req := newSIPRequest(sip.BYE, "call-bye-outbound")
	req.AppendHeader(sip.NewHeader("Reason", `Q.850;cause=16;text="Normal call clearing"`))
	tx := newTestServerTx()

	s.handleBye(req, tx)

	assert.True(t, session.IsEnded())
	select {
	case <-session.ByeReceived():
	default:
		t.Fatalf("expected ByeReceived signal")
	}
	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 200, tx.lastStatus())
	assert.True(t, onByeCalled)
	assert.False(t, disconnectCalled, "remote outbound BYE must not trigger local BYE")

	metadata := session.GetDisconnectMetadata()
	assert.Equal(t, DisconnectReasonNormalClearing, metadata.Reason)
	assert.Equal(t, 16, metadata.ProviderStatusCode)
	assert.Equal(t, "Normal call clearing", metadata.Text)
	_, ok := s.GetSession("call-bye-outbound")
	assert.False(t, ok, "remote outbound BYE should remove the ended session")
}

func TestSIPCommand_BYE_InboundSessionWithoutDialogReturns481(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.BYE, "call-bye-missing-dialog")
	tx := newTestServerTx()

	session := newTestSession(t, "call-bye-missing-dialog", CallDirectionInbound)
	session.SetState(CallStateConnected)
	s.sessions["call-bye-missing-dialog"] = session
	onByeCalled := false
	s.SetOnBye(func(*Session) error {
		onByeCalled = true
		return nil
	})

	s.handleBye(req, tx)

	assert.False(t, session.IsEnded())
	select {
	case <-session.ByeReceived():
		t.Fatalf("unexpected ByeReceived signal for invalid inbound BYE")
	default:
	}
	assert.False(t, onByeCalled)
	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 481, tx.lastStatus())
}

func TestSIPCommand_BYE_UnknownSession_Returns481(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.BYE, "call-bye-unknown")
	tx := newTestServerTx()

	s.handleBye(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 481, tx.lastStatus())
}

func TestSIPCommand_CANCEL_ExistingSession_EndsAndCallbacks(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.CANCEL, "call-cancel-1")
	tx := newTestServerTx()

	session := newTestSession(t, "call-cancel-1", CallDirectionInbound)
	s.registerSession(session, "call-cancel-1")

	cancelCalled := false
	s.onCancel = func(_ *Session) error {
		cancelCalled = true
		return nil
	}

	s.handleCancel(req, tx)

	assert.True(t, cancelCalled)
	assert.True(t, session.IsEnded())
	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 200, tx.lastStatus())
	_, ok := s.sessions["call-cancel-1"]
	assert.False(t, ok)
}

func TestSIPCommand_CANCEL_PendingInvite_Sends200And487(t *testing.T) {
	s := newServerForCommandTests(t)

	inviteReq := newSIPRequest(sip.INVITE, "call-cancel-pending")
	inviteTx := newTestServerTx()
	s.setPendingInvite(inboundInviteKey{callID: "call-cancel-pending", fromTag: "fromtag"}, inviteReq, inviteTx)

	cancelReq := newSIPRequest(sip.CANCEL, "call-cancel-pending")
	cancelTx := newTestServerTx()

	s.handleCancel(cancelReq, cancelTx)

	require.NotEmpty(t, cancelTx.responses)
	assert.Equal(t, 200, cancelTx.lastStatus())
	require.NotEmpty(t, inviteTx.responses)
	assert.Equal(t, 487, inviteTx.lastStatus())
}

func TestSIPCommand_REINVITE_InvalidSDPRejects400(t *testing.T) {
	s := newServerForCommandTests(t)
	session := registerConnectedInboundDialogSession(t, s, "call-reinvite-invalid")
	req := newInboundDialogSDPRequest(t, session, sip.INVITE, "bad-sdp")
	tx := newTestServerTx()

	s.handleInvite(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 400, tx.lastStatus())
}

func TestSIPCommand_REINVITE_UnsupportedCodecRejects488(t *testing.T) {
	s := newServerForCommandTests(t)
	session := registerConnectedInboundDialogSession(t, s, "call-reinvite-unsupported")
	req := newInboundDialogSDPRequest(t, session, sip.INVITE, unsupportedInboundOfferSDP())
	tx := newTestServerTx()

	s.handleInvite(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 488, tx.lastStatus())
}

func TestSIPCommand_REINVITE_HoldSDPAccepted(t *testing.T) {
	s := newServerForCommandTests(t)
	session := registerConnectedInboundDialogSession(t, s, "call-reinvite-hold")
	req := newInboundDialogSDPRequest(t, session, sip.INVITE, inboundOfferSDPWithMedia("0.0.0.0", 19000, "0 101"))
	tx := newAckableTestServerTx()
	ackRequest := newInboundDialogRequest(t, session, sip.ACK)
	tx.PushACK(ackRequest)

	s.handleInvite(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 200, tx.lastStatus())
	assert.Equal(t, "127.0.0.1:19000", session.GetInfo().RemoteRTPAddress)
	assert.Equal(t, uint64(1), session.ReInviteACKCount())
}

func TestSIPCommand_REINVITE_ACKTimeoutReturnsAfter200(t *testing.T) {
	s := newServerForCommandTests(t)
	s.inboundACKTimeout = time.Millisecond
	session := registerConnectedInboundDialogSession(t, s, "call-reinvite-no-ack")
	req := newInboundDialogSDPRequest(t, session, sip.INVITE, inboundOfferSDPWithMedia("0.0.0.0", 19000, "0 101"))
	tx := newAckableTestServerTx()

	s.handleInvite(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 200, tx.lastStatus())
	assert.Equal(t, CallStateConnected, session.GetState())
	assert.False(t, session.HasReInviteACKPending())
	assert.Equal(t, uint64(0), session.ReInviteACKCount())

	s.handleAck(newACKRequest("call-reinvite-no-ack"), newTestServerTx())

	assert.Equal(t, uint64(0), session.ReInviteACKCount())
}

func TestSIPCommand_REINVITE_ExistingSessionWithoutDialogReturns481(t *testing.T) {
	s := newServerForCommandTests(t)
	registerConnectedInboundSession(t, s, "call-reinvite-missing-dialog")
	req := newDialogSDPRequest(sip.INVITE, "call-reinvite-missing-dialog", validInboundOfferSDP())
	tx := newTestServerTx()

	s.handleInvite(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 481, tx.lastStatus())
}

func TestSIPCommand_UPDATE_UnknownSessionReturns481(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newDialogSDPRequest(sip.UPDATE, "call-update-unknown", validInboundOfferSDP())
	tx := newTestServerTx()

	s.handleUpdate(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 481, tx.lastStatus())
}

func TestSIPCommand_UPDATE_ExistingSessionWithoutDialogReturns481(t *testing.T) {
	s := newServerForCommandTests(t)
	registerConnectedInboundSession(t, s, "call-update-missing-dialog")
	req := newDialogSDPRequest(sip.UPDATE, "call-update-missing-dialog", validInboundOfferSDP())
	tx := newTestServerTx()

	s.handleUpdate(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 481, tx.lastStatus())
}

func TestSIPCommand_UPDATE_InvalidSDPRejects400(t *testing.T) {
	s := newServerForCommandTests(t)
	session := registerConnectedInboundDialogSession(t, s, "call-update-invalid")
	req := newInboundDialogSDPRequest(t, session, sip.UPDATE, "bad-sdp")
	tx := newTestServerTx()

	s.handleUpdate(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 400, tx.lastStatus())
}

func TestSIPCommand_UPDATE_UnsupportedContentTypeRejects415(t *testing.T) {
	s := newServerForCommandTests(t)
	session := registerConnectedInboundDialogSession(t, s, "call-update-content-type")
	req := newInboundDialogRequest(t, session, sip.UPDATE)
	req.AppendHeader(sip.NewHeader("Content-Type", "application/json"))
	req.SetBody([]byte(validInboundOfferSDP()))
	tx := newTestServerTx()

	s.handleUpdate(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 415, tx.lastStatus())
}

func TestSIPCommand_UPDATE_UnsupportedCodecRejects488(t *testing.T) {
	s := newServerForCommandTests(t)
	session := registerConnectedInboundDialogSession(t, s, "call-update-unsupported")
	req := newInboundDialogSDPRequest(t, session, sip.UPDATE, unsupportedInboundOfferSDP())
	tx := newTestServerTx()

	s.handleUpdate(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 488, tx.lastStatus())
}

func TestRegisterSession_RemoveHappensOnEndNotDisconnect(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-lifecycle-1", CallDirectionOutbound)

	s.registerSession(session, "call-lifecycle-1")
	session.Disconnect()
	_, exists := s.GetSession("call-lifecycle-1")
	assert.True(t, exists, "session should not be removed on Disconnect")

	session.End()
	_, exists = s.GetSession("call-lifecycle-1")
	assert.False(t, exists, "session should be removed after End")
}

func TestCallLifecycle_InvalidTransitionRejected(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-lifecycle-invalid", CallDirectionInbound)
	s.registerSession(session, "call-lifecycle-invalid")

	require.True(t, s.setCallState(session, CallStateRinging, "test_ringing"))
	require.True(t, s.setCallState(session, CallStateConnected, "test_connected"))
	require.False(t, s.setCallState(session, CallStateRinging, "invalid_back_to_ringing"))
	assert.Equal(t, CallStateConnected, session.GetState())
}

func TestCallLifecycle_FailFromRinging(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-lifecycle-failed", CallDirectionOutbound)
	s.registerSession(session, "call-lifecycle-failed")

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonOutboundProgressRinging))
	require.NoError(t, s.FailCall(session, LifecycleReasonOutboundWaitAnswerFailed, assert.AnError))

	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateFailed, session.GetState())
}

func TestCallLifecycle_CancelBeforeAnswerDoesNotDisconnect(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-lifecycle-cancelled", CallDirectionOutbound)
	s.registerSession(session, "call-lifecycle-cancelled")

	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})
	preAnswerCancelCalls := 0
	session.SetOnPreAnswerCancel(func() {
		preAnswerCancelCalls++
	})

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonOutboundProgressRinging))
	require.NoError(t, s.EndCallWithReason(session, LifecycleReasonEndCall))

	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateCancelled, session.GetState())
	assert.Equal(t, 1, preAnswerCancelCalls)
	assert.False(t, disconnectCalled)

	require.NoError(t, s.CancelCall(session, LifecycleReasonOutboundCancelledBeforeAnswer))
	assert.Equal(t, 1, preAnswerCancelCalls)

	s.mu.RLock()
	_, lifecycleExists := s.lifecycles[session.GetCallID()]
	s.mu.RUnlock()
	assert.False(t, lifecycleExists)
}

func TestCallLifecycle_CancelConnectedOutboundRoutesToBye(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-lifecycle-connected-cancel", CallDirectionOutbound)
	s.registerSession(session, "call-lifecycle-connected-cancel")

	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})
	preAnswerCancelCalls := 0
	session.SetOnPreAnswerCancel(func() {
		preAnswerCancelCalls++
	})

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonOutboundProgressRinging))
	require.True(t, s.TransitionCall(session, CallStateConnected, LifecycleReasonOutboundACKSent))
	session.SetOutboundDialogPhase(OutboundDialogPhaseConfirmed)
	require.NoError(t, s.CancelCall(session, LifecycleReasonOutboundCancelledBeforeAnswer))

	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateEnded, session.GetState())
	assert.True(t, disconnectCalled)
	assert.Equal(t, 0, preAnswerCancelCalls)
}

func TestCallLifecycle_FailAfterAnswerDisconnects(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-lifecycle-answer-failed", CallDirectionOutbound)
	s.registerSession(session, "call-lifecycle-answer-failed")

	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})
	preAnswerCancelCalls := 0
	session.SetOnPreAnswerCancel(func() {
		preAnswerCancelCalls++
	})

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonOutboundProgressRinging))
	session.SetOutboundDialogPhase(OutboundDialogPhaseAnswered)
	require.NoError(t, s.FailCall(session, LifecycleReasonOutboundAnswerSDPFailed, assert.AnError))

	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateFailed, session.GetState())
	assert.True(t, disconnectCalled)
	assert.Equal(t, 0, preAnswerCancelCalls)
}

func TestCallLifecycle_ConnectedEndDisconnects(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-lifecycle-ended", CallDirectionOutbound)
	s.registerSession(session, "call-lifecycle-ended")

	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonOutboundProgressRinging))
	require.True(t, s.TransitionCall(session, CallStateConnected, LifecycleReasonOutboundACKSent))
	require.NoError(t, s.EndCallWithReason(session, LifecycleReasonEndCall))

	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateEnded, session.GetState())
	assert.True(t, disconnectCalled)
}

func TestInboundLifecycle_PreAnswerCancelDoesNotDisconnect(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-inbound-cancelled", CallDirectionInbound)
	s.registerSession(session, "call-inbound-cancelled")

	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonInboundInviteRinging))
	require.NoError(t, s.CancelInboundCall(session, LifecycleReasonCancelReceived))

	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateCancelled, session.GetState())
	assert.False(t, disconnectCalled, "pre-answer inbound CANCEL must not send BYE")
}

func TestInboundLifecycle_PreAnswerFailureDoesNotDisconnect(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-inbound-failed-pre-answer", CallDirectionInbound)
	s.registerSession(session, "call-inbound-failed-pre-answer")

	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonInboundInviteRinging))
	require.NoError(t, s.FailInboundCall(session, LifecycleReasonInboundInviteFailed, assert.AnError))

	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateFailed, session.GetState())
	assert.False(t, disconnectCalled, "pre-answer inbound failure must not send BYE")
}

func TestInboundLifecycle_ConnectedEndDisconnects(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-inbound-ended", CallDirectionInbound)
	s.registerSession(session, "call-inbound-ended")

	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonInboundInviteRinging))
	require.True(t, s.TransitionCall(session, CallStateConnected, LifecycleReasonInboundInviteACKReceived))
	require.NoError(t, s.EndInboundCall(session, LifecycleReasonEndCall))

	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateEnded, session.GetState())
	assert.True(t, disconnectCalled, "local inbound end must send BYE after answer")
}

func TestInboundLifecycle_ConnectedFailureDisconnects(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-inbound-failed-after-answer", CallDirectionInbound)
	s.registerSession(session, "call-inbound-failed-after-answer")

	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonInboundInviteRinging))
	require.True(t, s.TransitionCall(session, CallStateConnected, LifecycleReasonInboundInviteACKReceived))
	require.NoError(t, s.FailInboundCall(session, LifecycleReasonPipelineSetupFailed, assert.AnError))

	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateFailed, session.GetState())
	assert.True(t, disconnectCalled, "post-answer inbound failure must send BYE")
}

func TestInboundLifecycle_ServerStopDisconnectsAnsweredCall(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-inbound-server-stop", CallDirectionInbound)
	s.registerSession(session, "call-inbound-server-stop")

	disconnectCalled := false
	session.SetOnDisconnect(func(*Session) {
		disconnectCalled = true
	})

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonInboundInviteRinging))
	require.True(t, s.TransitionCall(session, CallStateConnected, LifecycleReasonInboundInviteACKReceived))
	require.NoError(t, s.EndInboundCall(session, LifecycleReasonServerStop))

	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateEnded, session.GetState())
	assert.True(t, disconnectCalled, "server stop must send BYE for answered inbound calls")
}

func TestServerStopEndsSessionsThroughSessionCleanup(t *testing.T) {
	s := newServerForCommandTests(t)
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel
	s.state.Store(int32(ServerStateRunning))

	session := newTestSession(t, "call-server-stop-rtp", CallDirectionInbound)
	session.SetLocalRTP("127.0.0.1", 19000)
	s.registerSession(session, "call-server-stop-rtp")

	require.True(t, s.TransitionCall(session, CallStateConnected, LifecycleReasonInboundInviteACKReceived))
	s.Stop()

	assert.True(t, session.IsEnded())
	assert.Equal(t, 0, s.SessionCount())
}

func TestCallLifecycle_TransferSequence(t *testing.T) {
	s := newServerForCommandTests(t)
	session := newTestSession(t, "call-lifecycle-transfer", CallDirectionInbound)
	s.registerSession(session, "call-lifecycle-transfer")

	require.True(t, s.TransitionCall(session, CallStateRinging, LifecycleReasonInboundInviteRinging))
	require.True(t, s.TransitionCall(session, CallStateConnected, LifecycleReasonInboundInviteAnswered))
	require.True(t, s.TransitionCall(session, CallStateTransferring, LifecycleReasonBridgeTransferStarted))
	require.True(t, s.TransitionCall(session, CallStateBridgeConnected, LifecycleReasonBridgeMediaConnected))
	require.True(t, s.TransitionCall(session, CallStateConnected, LifecycleReasonTransferModeEnded))

	assert.Equal(t, CallStateConnected, session.GetState())
}

func TestSIPCommand_CANCEL_UnknownSession_Returns481(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.CANCEL, "call-cancel-unknown")
	tx := newTestServerTx()

	s.handleCancel(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 481, tx.lastStatus())
}

func TestSIPCommand_CANCEL_ConnectedSession_Returns481(t *testing.T) {
	s := newServerForCommandTests(t)
	req := newSIPRequest(sip.CANCEL, "call-cancel-connected")
	tx := newTestServerTx()

	session := newTestSession(t, "call-cancel-connected", CallDirectionInbound)
	session.SetState(CallStateConnected)
	s.registerSession(session, "call-cancel-connected")

	s.handleCancel(req, tx)

	require.NotEmpty(t, tx.responses)
	assert.Equal(t, 481, tx.lastStatus())
	assert.False(t, session.IsEnded())
	_, ok := s.GetSession("call-cancel-connected")
	assert.True(t, ok)
}

func TestSIPCommand_REGISTER_OPTIONS_INFO_Return200(t *testing.T) {
	s := newServerForCommandTests(t)

	cases := []struct {
		name   string
		method sip.RequestMethod
		run    func(*sip.Request, sip.ServerTransaction)
	}{
		{"REGISTER", sip.REGISTER, s.handleRegister},
		{"OPTIONS", sip.OPTIONS, s.handleOptions},
		{"INFO", sip.INFO, s.handleInfo},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := newSIPRequest(tc.method, "call-"+string(tc.method))
			tx := newTestServerTx()
			tc.run(req, tx)
			require.NotEmpty(t, tx.responses)
			assert.Equal(t, 200, tx.lastStatus())
		})
	}
}

func TestSIPCommand_UnknownMethodRouting(t *testing.T) {
	s := newServerForCommandTests(t)

	t.Run("in-dialog unknown returns 200", func(t *testing.T) {
		callID := "call-unknown-in-dialog"
		s.sessions[callID] = newTestSession(t, callID, CallDirectionInbound)

		req := newSIPRequest(sip.RequestMethod("PUBLISH"), callID)
		tx := newTestServerTx()
		s.handleUnknownRequest(req, tx)

		require.NotEmpty(t, tx.responses)
		assert.Equal(t, 200, tx.lastStatus())
	})

	t.Run("out-of-dialog SUBSCRIBE returns 489", func(t *testing.T) {
		req := newSIPRequest(sip.SUBSCRIBE, "call-unknown-subscribe")
		tx := newTestServerTx()
		s.handleUnknownRequest(req, tx)

		require.NotEmpty(t, tx.responses)
		assert.Equal(t, 489, tx.lastStatus())
	})

	t.Run("out-of-dialog other returns 405", func(t *testing.T) {
		req := newSIPRequest(sip.RequestMethod("PUBLISH"), "call-unknown-publish")
		tx := newTestServerTx()
		s.handleUnknownRequest(req, tx)

		require.NotEmpty(t, tx.responses)
		assert.Equal(t, 405, tx.lastStatus())
	})
}
