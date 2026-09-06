// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/emiago/sipgo/sip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboundCall_InvalidSDPRejectsWithoutSession(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newSIPRequest(sip.INVITE, "inbound-invalid-sdp")
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 400, transaction.lastStatus())
	_, exists := server.GetSession("inbound-invalid-sdp")
	assert.False(t, exists)
}

func TestInboundCall_InvalidIdentityRejectsWithoutSession(t *testing.T) {
	cases := []struct {
		name             string
		callID           string
		removeHeader     string
		removeFromTag    bool
		emptyFromAddress bool
		emptyToAddress   bool
	}{
		{name: "missing call id", callID: "inbound-missing-call-id", removeHeader: "Call-ID"},
		{name: "missing from", callID: "inbound-missing-from", removeHeader: "From"},
		{name: "missing from tag", callID: "inbound-missing-from-tag", removeFromTag: true},
		{name: "missing to", callID: "inbound-missing-to", removeHeader: "To"},
		{name: "empty from address", callID: "inbound-empty-from-address", emptyFromAddress: true},
		{name: "empty to address", callID: "inbound-empty-to-address", emptyToAddress: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := newServerForCommandTests(t)
			request := newInboundInviteRequest(tc.callID)
			for request.RemoveHeader(tc.removeHeader) {
			}
			if tc.removeFromTag && request.From() != nil && request.From().Params != nil {
				delete(request.From().Params, "tag")
			}
			if tc.emptyFromAddress {
				request.From().Address = sip.Uri{Scheme: "sip"}
			}
			if tc.emptyToAddress {
				request.To().Address = sip.Uri{Scheme: "sip"}
			}
			transaction := newTestServerTx()

			server.handleInvite(request, transaction)

			require.NotEmpty(t, transaction.responses)
			assert.Equal(t, 400, transaction.lastStatus())
			if tc.removeHeader != "Call-ID" {
				_, exists := server.GetSession(tc.callID)
				assert.False(t, exists)
			}
		})
	}
}

func TestInboundInviteIdentityFromRequestPreservesRoutingAndPartyIdentities(t *testing.T) {
	request := newInboundInviteRequest("identity-snapshot")
	request.Recipient = sip.Uri{Scheme: "sip", User: "agent-42", Host: "sip.rapida.ai"}
	request.From().Address = sip.Uri{Scheme: "sip", User: "alice", Host: "example.com"}
	request.To().Address = sip.Uri{Scheme: "sips", User: "support", Host: "example.net"}

	identity, ok := inboundInviteIdentityFromRequest(request)

	require.True(t, ok)
	assert.Equal(t, request.Recipient.String(), identity.requestURI)
	assert.Empty(t, identity.callAddress.From)
	assert.Empty(t, identity.callAddress.To)
	assert.Equal(t, request.From().Address.String(), identity.callAddress.FromURI)
	assert.Equal(t, request.To().Address.String(), identity.callAddress.ToURI)
}

func TestInboundMediaSDPConfigUsesLocalPacketization(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-answer-ptime")
	request.SetBody([]byte(strings.Replace(
		validInboundOfferSDP(),
		"a=sendrecv\r\n",
		"a=sendrecv\r\na=ptime:30\r\n",
		1,
	)))
	mediaOffer, failure := newInboundMediaOffer(
		server,
		request,
		"inbound INVITE",
		LifecycleReasonInboundInviteFailed,
		false,
	)
	require.Nil(t, failure)
	media := newInboundMedia(server, newTestSession(t, "inbound-answer-ptime", CallDirectionInbound), mediaOffer)
	media.externalIP = "127.0.0.1"
	media.localRTPPort = 20000

	sdpConfig := media.SDPConfig()

	require.NotNil(t, sdpConfig)
	assert.Equal(t, sdpDefaultPTimeMS, sdpConfig.PTime)
	assert.Equal(t, 30*time.Millisecond, mediaOffer.sdpInfo.PacketizationDuration())
}

func TestNewCallAddressAcceptsOnlyPhoneFromUser(t *testing.T) {
	tests := []struct {
		name     string
		fromUser string
		expected string
	}{
		{name: "national", fromUser: "07249994778", expected: "07249994778"},
		{name: "international", fromUser: "+447249994778", expected: "+447249994778"},
		{name: "surrounding ascii whitespace", fromUser: " \t+15551234567\r\n", expected: ""},
		{name: "empty", fromUser: "", expected: ""},
		{name: "plus only", fromUser: "+", expected: ""},
		{name: "multiple plus", fromUser: "++1555", expected: ""},
		{name: "embedded whitespace", fromUser: "+1555 1234", expected: ""},
		{name: "punctuation", fromUser: "+1-555-1234", expected: ""},
		{name: "uri parameter", fromUser: "+1555;user=phone", expected: ""},
		{name: "letters", fromUser: "agent-42", expected: ""},
		{name: "unicode digits", fromUser: "１２３４", expected: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := newInboundInviteRequest("phone-grammar")
			request.From().Address.User = test.fromUser
			request.To().Address.User = "+15557654321"

			address := NewCallAddress(request)

			assert.Equal(t, test.expected, address.From)
			assert.Empty(t, address.To)
			assert.Equal(t, request.From().Address.String(), address.FromURI)
			assert.Equal(t, request.To().Address.String(), address.ToURI)
		})
	}
}

func TestNewCallAddressCapturesHeadersWithoutCredentials(t *testing.T) {
	request := newInboundInviteRequest("identity-headers")
	request.AppendHeader(sip.NewHeader("X-Original-Called-Number", "+14155550200"))
	request.AppendHeader(sip.NewHeader("X-Original-Called-Number", "+14155550300"))
	request.AppendHeader(sip.NewHeader("Authorization", "secret"))
	request.AppendHeader(sip.NewHeader("Proxy-Authorization", "proxy-secret"))

	address := NewCallAddress(request)

	assert.Empty(t, address.From)
	assert.Empty(t, address.To)
	assert.Equal(t, request.From().Address.String(), address.FromURI)
	assert.Equal(t, request.To().Address.String(), address.ToURI)
	assert.Equal(t, "+14155550200,+14155550300", address.Headers["x-original-called-number"])
	assert.NotContains(t, address.Headers, "authorization")
	assert.NotContains(t, address.Headers, "proxy-authorization")
}

func TestNewCallAddressHandlesNilRequest(t *testing.T) {
	assert.Equal(t, CallAddress{}, NewCallAddress(nil))
}

func TestInboundCall_ConfigRejectDoesNotCreateSession(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		return &SIPError{Code: 403, Message: "forbidden", Err: ErrAuthRequired}
	}})
	request := newInboundInviteRequest("inbound-config-reject")
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 403, transaction.lastStatus())
	_, exists := server.GetSession("inbound-config-reject")
	assert.False(t, exists)
}

func TestInboundCall_ReplaysRejectedInviteWithoutMiddleware(t *testing.T) {
	server := newServerForCommandTests(t)
	middlewareCalls := 0
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		middlewareCalls++
		return &SIPError{Code: 403, Message: "forbidden", Err: ErrAuthRequired}
	}})

	firstRequest := newInboundInviteRequest("inbound-rejected-retry")
	firstTransaction := newTestServerTx()
	server.handleInvite(firstRequest, firstTransaction)

	secondRequest := newInboundInviteRequest("inbound-rejected-retry")
	secondTransaction := newTestServerTx()
	server.handleInvite(secondRequest, secondTransaction)

	require.NotEmpty(t, firstTransaction.responses)
	require.NotEmpty(t, secondTransaction.responses)
	assert.Equal(t, 403, firstTransaction.lastStatus())
	assert.Equal(t, 403, secondTransaction.lastStatus())
	assert.Equal(t, 1, middlewareCalls)
	_, exists := server.GetSession("inbound-rejected-retry")
	assert.False(t, exists)
}

func TestInboundCall_MiddlewareErrorRejectsWithoutSession(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		return errors.New("resolver unavailable")
	}})
	request := newInboundInviteRequest("inbound-config-error")
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 500, transaction.lastStatus())
	_, exists := server.GetSession("inbound-config-error")
	assert.False(t, exists)
}

func TestInboundCall_InvalidSessionConfigRejectsWithoutSession(t *testing.T) {
	server := newServerForCommandTests(t)
	invalidConfig := bridgeTestConfig()
	invalidConfig.RTPPortRangeStart = 20000
	invalidConfig.RTPPortRangeEnd = 10000
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = invalidConfig
		return nil
	}})
	request := newInboundInviteRequest("inbound-session-config-invalid")
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 500, transaction.lastStatus())
	_, exists := server.GetSession("inbound-session-config-invalid")
	assert.False(t, exists)
}

func TestInboundCall_DialogCreationFailureRespondsAndFailsSession(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	var failedSession *Session
	server.SetOnError(func(session *Session, _ error) {
		failedSession = session
	})
	request := newInboundInviteRequest("inbound-dialog-create-failed")
	for request.RemoveHeader("Contact") {
	}
	transaction := newActiveTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 500, transaction.lastStatus())
	require.NotNil(t, failedSession)
	assert.True(t, failedSession.IsEnded())
	assert.Equal(t, CallStateFailed, failedSession.GetState())
	assertSessionMetadata(t, failedSession, "sip.failure_class", string(inboundFailureDialog))
	assertSessionMetadata(t, failedSession, "sip.failure_response_class", string(inboundFailureDialog))
	assertSessionMetadata(t, failedSession, "sip.sli_result", string(CallTerminationServerError))
	assertSessionMetadata(t, failedSession, "sip.sli_reason", "inbound_dialog")
	assertSessionMetadata(t, failedSession, "sip.failure_status_code", 500)
	_, exists := server.GetSession("inbound-dialog-create-failed")
	assert.False(t, exists)
}

func TestInboundCall_UnsupportedCodecRejectsWithoutFallback(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-unsupported-codec")
	request.SetBody([]byte(unsupportedInboundOfferSDP()))
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 488, transaction.lastStatus())
	_, exists := server.GetSession("inbound-unsupported-codec")
	assert.False(t, exists)
}

func TestInboundCall_UnsupportedContentTypeRejects415(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-unsupported-content")
	for request.RemoveHeader("Content-Type") {
	}
	request.AppendHeader(sip.NewHeader("Content-Type", "application/json"))
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 415, transaction.lastStatus())
	_, exists := server.GetSession("inbound-unsupported-content")
	assert.False(t, exists)
}

func TestInboundCall_MissingContentTypeRejects415(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-missing-content")
	for request.RemoveHeader("Content-Type") {
	}
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 415, transaction.lastStatus())
	_, exists := server.GetSession("inbound-missing-content")
	assert.False(t, exists)
}

func TestInboundCall_InvalidRemoteRTPAddressRejects400(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-invalid-rtp-ip")
	request.SetBody([]byte(inboundOfferSDPWithMedia("not-an-ip", 19000, "0 101")))
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 400, transaction.lastStatus())
	_, exists := server.GetSession("inbound-invalid-rtp-ip")
	assert.False(t, exists)
}

func TestInboundCall_MissingRemoteRTPAddressRejects400(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-missing-rtp-ip")
	request.SetBody([]byte(inboundOfferSDPWithoutConnection()))
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 400, transaction.lastStatus())
	_, exists := server.GetSession("inbound-missing-rtp-ip")
	assert.False(t, exists)
}

func TestInboundCall_DisabledRemoteRTPAddressRejects488(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-disabled-rtp-ip")
	request.SetBody([]byte(inboundOfferSDPWithMedia("0.0.0.0", 19000, "0 101")))
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 488, transaction.lastStatus())
	_, exists := server.GetSession("inbound-disabled-rtp-ip")
	assert.False(t, exists)
}

func TestInboundCall_InvalidRemoteRTPPortRejects400(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-invalid-rtp-port")
	request.SetBody([]byte(inboundOfferSDPWithMedia("127.0.0.1", 70000, "0 101")))
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 400, transaction.lastStatus())
	_, exists := server.GetSession("inbound-invalid-rtp-port")
	assert.False(t, exists)
}

func TestInboundCall_MissingRemoteRTPPortRejects400(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-missing-rtp-port")
	request.SetBody([]byte(inboundOfferSDPWithRawMedia("127.0.0.1", "m=audio RTP/AVP 0 101")))
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 400, transaction.lastStatus())
	_, exists := server.GetSession("inbound-missing-rtp-port")
	assert.False(t, exists)
}

func TestInboundCall_NoPayloadTypesRejects488(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-no-payloads")
	request.SetBody([]byte(inboundOfferSDPWithMedia("127.0.0.1", 19000, "")))
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 488, transaction.lastStatus())
	_, exists := server.GetSession("inbound-no-payloads")
	assert.False(t, exists)
}

func TestInboundCall_RTPBindFailureEndsLifecycle(t *testing.T) {
	server := newServerForCommandTests(t)
	server.rtpPortRangeStart = 0
	server.rtpPortRangeEnd = 0
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	request := newInboundInviteRequest("inbound-rtp-failed")
	transaction := newActiveTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 503, transaction.lastStatus())
	_, exists := server.GetSession("inbound-rtp-failed")
	assert.False(t, exists)
}

func TestInboundCall_RTPHandlerCreationFailureEndsLifecycle(t *testing.T) {
	server := newServerForCommandTests(t)
	server.rtpPortRangeStart = 70000
	server.rtpPortRangeEnd = 70000
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	request := newInboundInviteRequest("inbound-rtp-handler-failed")
	transaction := newActiveTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 503, transaction.lastStatus())
	_, exists := server.GetSession("inbound-rtp-handler-failed")
	assert.False(t, exists)
}

func TestInboundCall_DialogSetupFailureSendsFinalResponse(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	request := newInboundInviteRequest("inbound-dialog-create-failed")
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 500, transaction.lastStatus())
	_, exists := server.GetSession("inbound-dialog-create-failed")
	assert.False(t, exists)
}

func TestInboundCall_TryingResponseFailureDoesNotStopTerminalResponse(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		return errors.New("resolver unavailable")
	}})
	request := newInboundInviteRequest("inbound-trying-failed")
	transaction := newFailingStatusServerTx(100)

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 500, transaction.lastStatus())
	assertSIPStatus(t, transaction.responses, 100)
	_, exists := server.GetSession("inbound-trying-failed")
	assert.False(t, exists)
}

func TestInboundCall_RingingResponseFailureEndsLifecycle(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	request := newInboundInviteRequest("inbound-ringing-failed")
	transaction := newFailingStatusServerTx(180)

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 500, transaction.lastStatus())
	assertSIPStatus(t, transaction.responses, 100)
	assertSIPStatus(t, transaction.responses, 180)
	_, exists := server.GetSession("inbound-ringing-failed")
	assert.False(t, exists)
}

func TestInboundCall_CancelBeforeSessionCreationStopsSetup(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		cancelRequest := newSIPRequest(sip.CANCEL, "inbound-cancel-before-session")
		cancelTransaction := newTestServerTx()
		server.handleCancel(cancelRequest, cancelTransaction)
		require.NotEmpty(t, cancelTransaction.responses)
		assert.Equal(t, 200, cancelTransaction.lastStatus())
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	request := newInboundInviteRequest("inbound-cancel-before-session")
	transaction := newActiveTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 487, transaction.lastStatus())
	assertNoSIPStatus(t, transaction.responses, 200)
	_, exists := server.GetSession("inbound-cancel-before-session")
	assert.False(t, exists)
}

func TestInboundCall_ApplicationReadyBeforeAnswerAndMediaStart(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.CallAddress.To = "+15557654321"
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	phaseOrder := make([]string, 0, 2)
	transaction := newActiveAckableTestServerTx()
	request := newInboundInviteRequest("inbound-ready-before-answer")
	expectedRequestURI := request.Recipient.String()
	expectedCallAddress := NewCallAddress(request)
	expectedCallAddress.To = "+15557654321"
	server.SetOnApplicationReady(func(session *Session, requestURI string, callAddress CallAddress) error {
		phaseOrder = append(phaseOrder, "application_ready")
		assert.Equal(t, 180, transaction.lastStatus())
		assert.Equal(t, InboundSetupPhaseMediaAllocated, session.GetInboundSetupPhase())
		assert.Equal(t, expectedRequestURI, requestURI)
		assert.Equal(t, expectedCallAddress, callAddress)
		return nil
	})
	server.SetOnInvite(func(session *Session, requestURI string, callAddress CallAddress) error {
		phaseOrder = append(phaseOrder, "call_start")
		assert.Equal(t, InboundSetupPhaseMediaFlowing, session.GetInboundSetupPhase())
		assert.Equal(t, expectedRequestURI, requestURI)
		assert.Equal(t, expectedCallAddress, callAddress)
		return nil
	})
	transaction.PushACK(newACKRequest("inbound-ready-before-answer"))

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 200, transaction.lastStatus())
	assert.Equal(t, []string{"application_ready", "call_start"}, phaseOrder)
	session, exists := server.GetSession("inbound-ready-before-answer")
	require.True(t, exists)
	assert.Equal(t, CallStateConnected, session.GetState())
	assert.Equal(t, InboundSetupPhaseMediaFlowing, session.GetInboundSetupPhase())
	timings := session.GetInboundSetupTimings()
	assert.False(t, timings.RingingSentAt.IsZero())
	assert.False(t, timings.AnsweredAt.IsZero())
}

func TestInboundCall_ReInviteDoesNotReplaceInitialPartyIdentities(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	callbackCount := 0
	var capturedFromIdentity string
	var capturedToIdentity string
	server.SetOnInvite(func(_ *Session, _ string, callAddress CallAddress) error {
		callbackCount++
		capturedFromIdentity = callAddress.FromURI
		capturedToIdentity = callAddress.ToURI
		return nil
	})

	initialRequest := newInboundInviteRequest("inbound-reinvite-identity")
	initialRequest.From().Address = sip.Uri{Scheme: "sip", User: "original-caller", Host: "example.com"}
	initialRequest.To().Address = sip.Uri{Scheme: "sip", User: "original-assistant", Host: "sip.rapida.ai"}
	initialTransaction := newActiveAckableTestServerTx()
	initialTransaction.PushACK(newACKRequest("inbound-reinvite-identity"))
	server.handleInvite(initialRequest, initialTransaction)

	require.Equal(t, 1, callbackCount)
	require.Equal(t, 200, initialTransaction.lastStatus())
	session, exists := server.GetSession("inbound-reinvite-identity")
	require.True(t, exists)

	reInvite := newInboundDialogSDPRequest(t, session, sip.INVITE, validInboundOfferSDP())
	reInvite.From().Address = sip.Uri{Scheme: "sip", User: "changed-caller", Host: "example.net"}
	reInvite.To().Address = sip.Uri{Scheme: "sip", User: "changed-assistant", Host: "sip.example.net"}
	reInviteTransaction := newTestServerTx()
	server.handleInvite(reInvite, reInviteTransaction)

	require.Equal(t, 200, reInviteTransaction.lastStatus())
	assert.Equal(t, 1, callbackCount)
	assert.Equal(t, "sip:original-caller@example.com", capturedFromIdentity)
	assert.Equal(t, "sip:original-assistant@sip.rapida.ai", capturedToIdentity)
}

func TestInboundCall_InviteWithReplacesUsesNewDialogPartyIdentities(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	replacedSession := registerConnectedInboundDialogSession(t, server, "replaced-dialog")

	callbackCount := 0
	var capturedRequestURI string
	var capturedFromIdentity string
	var capturedToIdentity string
	server.SetOnInvite(func(_ *Session, requestURI string, callAddress CallAddress) error {
		callbackCount++
		capturedRequestURI = requestURI
		capturedFromIdentity = callAddress.FromURI
		capturedToIdentity = callAddress.ToURI
		return nil
	})

	replacementRequest := newInboundInviteRequest("replacement-dialog")
	replacementRequest.Recipient = sip.Uri{Scheme: "sip", User: "agent-42", Host: "sip.rapida.ai"}
	replacementRequest.From().Address = sip.Uri{Scheme: "sip", User: "replacement-caller", Host: "example.net"}
	replacementRequest.To().Address = sip.Uri{Scheme: "sip", User: "replacement-assistant", Host: "sip.example.net"}
	replacementRequest.AppendHeader(sip.NewHeader("Replaces", replacedSession.GetCallID()+";to-tag=to-tag;from-tag=from-tag"))
	replacementTransaction := newActiveAckableTestServerTx()
	replacementTransaction.PushACK(newACKRequest("replacement-dialog"))

	server.handleInvite(replacementRequest, replacementTransaction)

	require.Equal(t, 200, replacementTransaction.lastStatus())
	assert.Equal(t, 1, callbackCount)
	assert.Equal(t, replacementRequest.Recipient.String(), capturedRequestURI)
	assert.Equal(t, "sip:replacement-caller@example.net", capturedFromIdentity)
	assert.Equal(t, "sip:replacement-assistant@sip.example.net", capturedToIdentity)
	_, replacedExists := server.GetSession("replaced-dialog")
	_, replacementExists := server.GetSession("replacement-dialog")
	assert.True(t, replacedExists)
	assert.True(t, replacementExists)
}

func TestInboundCall_REFERDoesNotReplaceInitialPartyIdentities(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	callbackCount := 0
	var capturedFromIdentity string
	var capturedToIdentity string
	server.SetOnInvite(func(_ *Session, _ string, callAddress CallAddress) error {
		callbackCount++
		capturedFromIdentity = callAddress.FromURI
		capturedToIdentity = callAddress.ToURI
		return nil
	})

	initialRequest := newInboundInviteRequest("inbound-refer-identity")
	initialRequest.From().Address = sip.Uri{Scheme: "sip", User: "original-caller", Host: "example.com"}
	initialRequest.To().Address = sip.Uri{Scheme: "sip", User: "original-assistant", Host: "sip.rapida.ai"}
	initialTransaction := newActiveAckableTestServerTx()
	initialTransaction.PushACK(newACKRequest("inbound-refer-identity"))
	server.handleInvite(initialRequest, initialTransaction)

	require.Equal(t, 1, callbackCount)
	session, exists := server.GetSession("inbound-refer-identity")
	require.True(t, exists)

	referRequest := newInboundDialogRequest(t, session, sip.REFER)
	referRequest.AppendHeader(sip.NewHeader("Refer-To", "<sip:transfer-target@example.net>"))
	referTransaction := newTestServerTx()
	server.handleRefer(referRequest, referTransaction)

	require.Equal(t, 603, referTransaction.lastStatus())
	assert.Equal(t, 1, callbackCount)
	assert.Equal(t, "sip:original-caller@example.com", capturedFromIdentity)
	assert.Equal(t, "sip:original-assistant@sip.rapida.ai", capturedToIdentity)
	assert.Equal(t, 1, server.SessionCount())
}

func TestInboundCall_ProvisionalResponsesBeforeAnswer(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	transaction := newActiveAckableTestServerTx()
	request := newInboundInviteRequest("inbound-provisional-order")
	transaction.PushACK(newACKRequest("inbound-provisional-order"))

	server.handleInvite(request, transaction)

	require.GreaterOrEqual(t, len(transaction.responses), 3)
	assert.Equal(t, 100, transaction.responses[0].StatusCode)
	assert.Equal(t, 180, transaction.responses[1].StatusCode)
	assert.Equal(t, 200, transaction.responses[2].StatusCode)
}

func TestInboundCall_StartsRTPAfterACK(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	transaction := newActiveAckableTestServerTx()
	request := newInboundInviteRequest("inbound-rtp-before-ack")
	done := make(chan struct{})

	go func() {
		defer close(done)
		server.handleInvite(request, transaction)
	}()

	require.Eventually(t, func() bool {
		return transaction.lastStatus() == 200
	}, time.Second, time.Millisecond)
	session, exists := server.GetSession("inbound-rtp-before-ack")
	require.True(t, exists)
	require.NotNil(t, session.GetRTPHandler())
	assert.Equal(t, InboundSetupPhaseAnswered, session.GetInboundSetupPhase())

	transaction.PushACK(newACKRequest("inbound-rtp-before-ack"))
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for inbound INVITE handler")
	}
	assert.Equal(t, InboundSetupPhaseMediaFlowing, session.GetInboundSetupPhase())
}

func TestInboundCall_RetransmitsRingingUntilAnswer(t *testing.T) {
	server := newServerForCommandTests(t)
	server.inboundRingingInterval = 5 * time.Millisecond
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	applicationReady := make(chan struct{})
	server.SetOnApplicationReady(func(_ *Session, _ string, _ CallAddress) error {
		<-applicationReady
		return nil
	})
	transaction := newActiveAckableTestServerTx()
	request := newInboundInviteRequest("inbound-ringing-retransmit")
	done := make(chan struct{})

	go func() {
		defer close(done)
		server.handleInvite(request, transaction)
	}()

	require.Eventually(t, func() bool {
		return transaction.statusCount(180) >= 2
	}, time.Second, time.Millisecond)
	assert.Equal(t, 0, transaction.statusCount(200))

	close(applicationReady)
	transaction.PushACK(newACKRequest("inbound-ringing-retransmit"))
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for inbound INVITE handler")
	}
	ringingCount := transaction.statusCount(180)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ringingCount, transaction.statusCount(180))
	assert.Equal(t, 200, transaction.lastStatus())
}

func TestInboundCall_WaitsForApplicationReadyBeforeAnswer(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	applicationReady := make(chan struct{})
	server.SetOnApplicationReady(func(_ *Session, _ string, _ CallAddress) error {
		<-applicationReady
		return nil
	})
	transaction := newActiveAckableTestServerTx()
	request := newInboundInviteRequest("inbound-waits-application-ready")
	done := make(chan struct{})

	go func() {
		defer close(done)
		server.handleInvite(request, transaction)
	}()

	require.Eventually(t, func() bool {
		return transaction.lastStatus() == 180
	}, time.Second, time.Millisecond)
	assertNoSIPStatus(t, transaction.responses, 200)

	close(applicationReady)
	transaction.PushACK(newACKRequest("inbound-waits-application-ready"))
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for inbound INVITE handler")
	}
	assert.Equal(t, 200, transaction.lastStatus())
}

func TestInboundCall_MinRingPolicyDelaysAnswer(t *testing.T) {
	server := newServerForCommandTests(t)
	minRingDuration := 25 * time.Millisecond
	config := bridgeTestConfig()
	config.InboundAnswerMode = InboundAnswerModeAfterMinRingDuration
	config.InboundMinRingDuration = minRingDuration
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = config
		return nil
	}})
	transaction := newActiveAckableTestServerTx()
	request := newInboundInviteRequest("inbound-min-ring")
	transaction.PushACK(newACKRequest("inbound-min-ring"))

	startedAt := time.Now()
	server.handleInvite(request, transaction)

	assert.GreaterOrEqual(t, time.Since(startedAt), minRingDuration)
	assert.Equal(t, 200, transaction.lastStatus())
}

func TestInboundCall_MinRingConfigRequiresDuration(t *testing.T) {
	server := newServerForCommandTests(t)
	config := bridgeTestConfig()
	config.InboundAnswerMode = InboundAnswerModeAfterMinRingDuration
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = config
		return nil
	}})
	transaction := newTestServerTx()
	request := newInboundInviteRequest("inbound-min-ring-requires-duration")

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 500, transaction.lastStatus())
	assertNoSIPStatus(t, transaction.responses, 200)
	session, exists := server.GetSession("inbound-min-ring-requires-duration")
	assert.False(t, exists)
	assert.Nil(t, session)
}

func TestInboundCall_AnswersAfterApplicationReadyWithoutAssistantAudio(t *testing.T) {
	server := newServerForCommandTests(t)
	config := bridgeTestConfig()
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = config
		return nil
	}})
	applicationReadyCalled := false
	server.SetOnApplicationReady(func(_ *Session, _ string, _ CallAddress) error {
		applicationReadyCalled = true
		return nil
	})
	transaction := newActiveAckableTestServerTx()
	request := newInboundInviteRequest("inbound-application-ready-no-audio")
	transaction.PushACK(newACKRequest("inbound-application-ready-no-audio"))

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.True(t, applicationReadyCalled)
	assert.Equal(t, 200, transaction.lastStatus())
	session, exists := server.GetSession("inbound-application-ready-no-audio")
	require.True(t, exists)
	timings := session.GetInboundSetupTimings()
	assert.True(t, timings.FirstAssistantAudioReadyAt.IsZero())
	assert.True(t, timings.FirstAssistantAudioSentAt.IsZero())
	metrics := session.GetInboundLatencyMetrics()
	assert.NotContains(t, metrics, "assistant_audio_ready_to_answer_ms")
	assert.NotContains(t, metrics, "answer_to_first_assistant_audio_sent_ms")
}

func TestInboundCall_UDPFinalResponseRetransmitsUntilACKTimeout(t *testing.T) {
	server := newServerForCommandTests(t)
	server.inboundACKTimeout = 35 * time.Millisecond
	server.inboundFinalResponseRetryInitial = 5 * time.Millisecond
	server.inboundFinalResponseRetryMax = 5 * time.Millisecond
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	transaction := newActiveAckableTestServerTx()
	request := newInboundInviteRequest("inbound-200-retry")

	server.handleInvite(request, transaction)

	okResponses := 0
	for _, response := range transaction.responses {
		if response.StatusCode == 200 {
			okResponses++
		}
	}
	assert.GreaterOrEqual(t, okResponses, 2)
	_, exists := server.GetSession("inbound-200-retry")
	assert.False(t, exists)
}

func TestInboundCall_RecordsInboundLatencyMetrics(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	transaction := newActiveAckableTestServerTx()
	request := newInboundInviteRequest("inbound-latency-metrics")
	transaction.PushACK(newACKRequest("inbound-latency-metrics"))

	server.handleInvite(request, transaction)

	session, exists := server.GetSession("inbound-latency-metrics")
	require.True(t, exists)
	metrics := session.GetInboundLatencyMetrics()
	assert.Contains(t, metrics, "invite_to_100_ms")
	assert.Contains(t, metrics, "invite_to_180_ms")
	assert.Contains(t, metrics, "180_to_200_ms")
	assert.Contains(t, metrics, "200_to_ack_ms")
}

func TestInboundCall_ApplicationReadinessFailureRejectsBeforeAnswer(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	server.SetOnApplicationReady(func(_ *Session, _ string, _ CallAddress) error {
		return errors.New("assistant not ready")
	})
	server.SetOnInvite(func(_ *Session, _ string, _ CallAddress) error {
		t.Fatal("onInvite should not run when application readiness fails")
		return nil
	})
	request := newInboundInviteRequest("inbound-readiness-failed")
	transaction := newActiveAckableTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 503, transaction.lastStatus())
	for _, response := range transaction.responses {
		assert.NotEqual(t, 200, response.StatusCode)
	}
	_, exists := server.GetSession("inbound-readiness-failed")
	assert.False(t, exists)
}

func TestInboundCall_ACKTimeoutCleansPreparedApplication(t *testing.T) {
	server := newServerForCommandTests(t)
	server.inboundACKTimeout = time.Millisecond
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	cleanupCount := 0
	var capturedSession *Session
	server.SetOnApplicationReady(func(session *Session, _ string, _ CallAddress) error {
		capturedSession = session
		return nil
	})
	server.SetOnApplicationCleanup(func(_ *Session) {
		cleanupCount++
	})
	server.SetOnInvite(func(_ *Session, _ string, _ CallAddress) error {
		t.Fatal("onInvite should not run when ACK timeout fails")
		return nil
	})
	request := newInboundInviteRequest("inbound-ack-timeout-cleanup")
	transaction := newActiveAckableTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 200, transaction.lastStatus())
	assert.Equal(t, 1, cleanupCount)
	require.NotNil(t, capturedSession)
	assert.True(t, capturedSession.IsEnded())
	assert.Equal(t, CallStateFailed, capturedSession.GetState())
	assertSessionMetadata(t, capturedSession, "sip.failure_class", string(inboundFailureNoACK))
	assertSessionMetadata(t, capturedSession, "sip.failure_response_class", string(inboundFailureDialog))
	assertSessionMetadata(t, capturedSession, "sip.sli_result", string(CallTerminationServerError))
	assertSessionMetadata(t, capturedSession, "sip.sli_reason", "inbound_no_ack")
	assertNoSessionMetadata(t, capturedSession, "sip.failure_status_code")
	_, exists := server.GetSession("inbound-ack-timeout-cleanup")
	assert.False(t, exists)
}

func TestInboundCall_CancelWhileWaitingForACKDoesNotSend487After200(t *testing.T) {
	server := newServerForCommandTests(t)
	server.SetMiddlewares([]Middleware{func(ctx *SIPRequestContext) error {
		ctx.Config = bridgeTestConfig()
		return nil
	}})
	request := newInboundInviteRequest("inbound-cancel-waiting-ack")
	transaction := newActiveAckableTestServerTx()
	done := make(chan struct{})

	go func() {
		defer close(done)
		server.handleInvite(request, transaction)
	}()

	require.Eventually(t, func() bool {
		return transaction.lastStatus() == 200
	}, time.Second, time.Millisecond)

	cancelTransaction := newTestServerTx()
	server.handleCancel(newSIPRequest(sip.CANCEL, "inbound-cancel-waiting-ack"), cancelTransaction)

	require.NotEmpty(t, cancelTransaction.responses)
	assert.Equal(t, 481, cancelTransaction.lastStatus())
	assertNoSIPStatus(t, transaction.responses, 487)
	transaction.PushACK(newACKRequest("inbound-cancel-waiting-ack"))

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for inbound INVITE handler")
	}
	session, exists := server.GetSession("inbound-cancel-waiting-ack")
	require.True(t, exists)
	assert.Equal(t, CallStateConnected, session.GetState())
}

func TestInboundCall_CancelAfterSessionRegistrationEndsLifecycle(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-cancel-registered")
	transaction := newActiveTestServerTx()
	inboundCall := NewInbound(server, request, transaction)

	loadInboundIdentity(t, inboundCall)
	loadInboundMediaOffer(t, inboundCall)
	inboundCall.resolvedConfig = inboundConfig{config: bridgeTestConfig()}
	createInboundSessionForTest(t, inboundCall)
	server.registerSession(inboundCall.session, inboundCall.identity.callID)
	require.True(t, server.setPendingInviteIfAbsent(inboundCall.inviteKey, request, transaction))
	server.markInviteCancelled(inboundCall.inviteKey)

	cancelled := server.terminatePendingInvite(inboundCall.inviteKey, 487)
	inboundCall.cleanupApplication()
	_ = server.CancelInboundCall(inboundCall.session, LifecycleReasonInviteCancelled)

	assert.True(t, cancelled)
	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 487, transaction.lastStatus())
	assertNoSIPStatus(t, transaction.responses, 200)
	assert.True(t, inboundCall.session.IsEnded())
	assert.Equal(t, CallStateCancelled, inboundCall.session.GetState())
	_, exists := server.GetSession("inbound-cancel-registered")
	assert.False(t, exists)
}

func TestInboundCall_CancelAfterRTPOwnershipEndsSession(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-cancel-rtp")
	transaction := newActiveTestServerTx()
	inboundCall := NewInbound(server, request, transaction)

	loadInboundIdentity(t, inboundCall)
	loadInboundMediaOffer(t, inboundCall)
	inboundCall.resolvedConfig = inboundConfig{config: bridgeTestConfig()}
	createInboundSessionForTest(t, inboundCall)
	inboundCall.session.SetLocalRTP("127.0.0.1", 19000)
	inboundCall.session.SetRTPHandler(&RTPHandler{})
	server.registerSession(inboundCall.session, inboundCall.identity.callID)
	require.True(t, server.setPendingInviteIfAbsent(inboundCall.inviteKey, request, transaction))
	server.markInviteCancelled(inboundCall.inviteKey)

	cancelled := server.terminatePendingInvite(inboundCall.inviteKey, 487)
	inboundCall.cleanupApplication()
	_ = server.CancelInboundCall(inboundCall.session, LifecycleReasonInviteCancelledBeforeAnswer)

	assert.True(t, cancelled)
	assertNoSIPStatus(t, transaction.responses, 200)
	assert.True(t, inboundCall.session.IsEnded())
	_, exists := server.GetSession("inbound-cancel-rtp")
	assert.False(t, exists)
}

func TestInboundCall_FinalResponseMediaTimeoutEndsCall(t *testing.T) {
	server := newServerForCommandTests(t)
	cleanupCalls := 0
	server.SetOnApplicationCleanup(func(*Session) {
		cleanupCalls++
	})

	callID := "inbound-final-response-media-timeout"
	session := newTestSession(t, callID, CallDirectionInbound)
	session.SetLocalRTP("127.0.0.1", 19000)
	session.SetRTPHandler(newTestRTPHandler())
	server.registerSession(session, callID)
	require.True(t, server.TransitionCall(session, CallStateRinging, LifecycleReasonInboundInviteRinging))
	require.True(t, session.MarkInitialACKReceived())
	require.True(t, server.TransitionCall(session, CallStateConnected, LifecycleReasonInboundInviteACKReceived))

	transaction := newActiveTestServerTx()
	inboundCall := &Inbound{
		server:      server,
		request:     newInboundInviteRequest(callID),
		transaction: transaction,
		identity:    inboundInviteIdentity{callID: callID},
		session:     session,
		dialog: &inboundDialog{
			finalResponseStarted: true,
		},
	}

	inboundCall.mediaTimeout()
	inboundCall.mediaTimeout()

	assert.Empty(t, transaction.responses)
	assert.True(t, session.IsEnded())
	assert.Equal(t, CallStateEnded, session.GetState())
	assert.Equal(t, 1, cleanupCalls)
	assertSessionMetadata(t, session, "sip.failure_class", string(inboundFailureMediaTimeout))
	assertSessionMetadata(t, session, "sip.failure_response_class", string(inboundFailureRTP))
	assertSessionMetadata(t, session, "sip.failure_reason", ErrRTPMediaTimeout.Error())
	assertSessionMetadata(t, session, "sip.sli_result", string(CallTerminationServerError))
	assertSessionMetadata(t, session, "sip.sli_reason", "inbound_media_timeout")
	assertSessionMetadata(t, session, "sip.failure_retryable", true)
	assertNoSessionMetadata(t, session, "sip.failure_status_code")
	_, exists := server.GetSession(callID)
	assert.False(t, exists)
}

func TestSIPCommand_CANCEL_ConnectedInboundReturns481(t *testing.T) {
	server := newServerForCommandTests(t)
	session := newTestSession(t, "inbound-cancel-connected", CallDirectionInbound)
	server.registerSession(session, "inbound-cancel-connected")
	require.True(t, server.TransitionCall(session, CallStateRinging, LifecycleReasonInboundInviteRinging))
	require.True(t, server.TransitionCall(session, CallStateConnected, LifecycleReasonInboundInviteAnswered))

	request := newSIPRequest(sip.CANCEL, "inbound-cancel-connected")
	transaction := newTestServerTx()

	server.handleCancel(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 481, transaction.lastStatus())
	assert.False(t, session.IsEnded())
}

type activeTestServerTx struct {
	*testServerTx
}

func newActiveTestServerTx() *activeTestServerTx {
	return &activeTestServerTx{testServerTx: newTestServerTx()}
}

func (t *activeTestServerTx) OnTerminate(_ sip.FnTxTerminate) bool {
	return true
}

func (t *activeTestServerTx) OnCancel(_ sip.FnTxCancel) bool {
	return true
}

type activeAckableTestServerTx struct {
	*activeTestServerTx
	acks chan *sip.Request
}

func newActiveAckableTestServerTx() *activeAckableTestServerTx {
	return &activeAckableTestServerTx{
		activeTestServerTx: newActiveTestServerTx(),
		acks:               make(chan *sip.Request, 2),
	}
}

func (t *activeAckableTestServerTx) Acks() <-chan *sip.Request {
	return t.acks
}

func (t *activeAckableTestServerTx) PushACK(req *sip.Request) {
	t.acks <- req
}

func assertNoSIPStatus(t *testing.T, responses []*sip.Response, statusCode int) {
	t.Helper()
	for _, response := range responses {
		assert.NotEqual(t, statusCode, response.StatusCode)
	}
}

func assertSIPStatus(t *testing.T, responses []*sip.Response, statusCode int) {
	t.Helper()
	for _, response := range responses {
		if response.StatusCode == statusCode {
			return
		}
	}
	t.Fatalf("expected SIP status %d in responses", statusCode)
}

func loadInboundIdentity(t *testing.T, inboundCall *Inbound) {
	t.Helper()
	require.NotNil(t, inboundCall.request)
	require.NotNil(t, inboundCall.request.CallID())
	require.NotNil(t, inboundCall.request.From())
	require.NotNil(t, inboundCall.request.To())
	inboundCall.identity = inboundInviteIdentity{
		callID:      inboundCall.request.CallID().Value(),
		fromTag:     "fromtag",
		requestURI:  inboundCall.request.Recipient.String(),
		callAddress: NewCallAddress(inboundCall.request),
	}
	inboundCall.inviteKey = inboundInviteKey{callID: inboundCall.identity.callID, fromTag: inboundCall.identity.fromTag}
}

func loadInboundMediaOffer(t *testing.T, inboundCall *Inbound) {
	t.Helper()
	mediaOffer, failure := newInboundMediaOffer(
		inboundCall.server,
		inboundCall.request,
		"inbound INVITE",
		LifecycleReasonInboundInviteFailed,
		false,
	)
	require.Nil(t, failure)
	inboundCall.mediaOffer = mediaOffer
}

func createInboundSessionForTest(t *testing.T, inboundCall *Inbound) {
	t.Helper()
	session, err := NewSession(inboundCall.server.ctx,
		WithSessionConfig(inboundCall.resolvedConfig.config),
		WithSessionDirection(CallDirectionInbound),
		WithSessionCallID(inboundCall.identity.callID),
		WithSessionCodec(inboundCall.mediaOffer.negotiatedCodec),
		WithSessionAuth(inboundCall.resolvedConfig.auth),
		WithSessionAssistant(inboundCall.resolvedConfig.assistant),
		WithSessionVaultCredential(inboundCall.resolvedConfig.vaultCredential),
	)
	require.NoError(t, err)
	inboundCall.session = session
}

func createInboundDialogForTest(t *testing.T, inboundCall *Inbound) {
	t.Helper()
	dialog, failure := newInboundDialog(
		inboundCall.server,
		inboundCall.session,
		inboundCall.request,
		inboundCall.transaction,
		inboundCall.inviteKey,
	)
	require.Nil(t, failure)
	inboundCall.dialog = dialog
	inboundCall.session.SetDialogServerSession(inboundCall.dialog.DialogSession())
}

func assertNoSessionMetadata(t *testing.T, session *Session, key string) {
	t.Helper()
	_, ok := session.GetMetadata(key)
	assert.False(t, ok, "metadata %s should be absent", key)
}

func unsupportedInboundOfferSDP() string {
	return inboundOfferSDPWithMedia("127.0.0.1", 19000, "18 101")
}

func inboundOfferSDPWithMedia(ip string, port int, payloadTypes string) string {
	mediaLine := "m=audio " + fmt.Sprintf("%d RTP/AVP", port)
	if payloadTypes != "" {
		mediaLine += " " + payloadTypes
	}
	return inboundOfferSDPWithRawMedia(ip, mediaLine)
}

func inboundOfferSDPWithRTCP(remoteRTPIPAddress string, remoteRTPPort int, remoteRTCPIPAddress string, remoteRTCPPort int) string {
	return "v=0\r\n" +
		"o=caller 1 1 IN IP4 127.0.0.1\r\n" +
		"s=call\r\n" +
		"c=IN IP4 " + remoteRTPIPAddress + "\r\n" +
		"t=0 0\r\n" +
		"m=audio " + fmt.Sprintf("%d RTP/AVP 0 101", remoteRTPPort) + "\r\n" +
		"a=rtpmap:0 PCMU/8000\r\n" +
		"a=rtpmap:101 telephone-event/8000\r\n" +
		"a=rtcp:" + fmt.Sprintf("%d IN IP4 %s", remoteRTCPPort, remoteRTCPIPAddress) + "\r\n" +
		"a=sendrecv\r\n"
}

func inboundOfferSDPWithoutConnection() string {
	return "v=0\r\n" +
		"o=caller 1 1 IN IP4 127.0.0.1\r\n" +
		"s=call\r\n" +
		"t=0 0\r\n" +
		"m=audio 19000 RTP/AVP 0 101\r\n" +
		"a=rtpmap:0 PCMU/8000\r\n" +
		"a=rtpmap:101 telephone-event/8000\r\n" +
		"a=sendrecv\r\n"
}

func inboundOfferSDPWithRawMedia(ip string, mediaLine string) string {
	return "v=0\r\n" +
		"o=caller 1 1 IN IP4 127.0.0.1\r\n" +
		"s=call\r\n" +
		"c=IN IP4 " + ip + "\r\n" +
		"t=0 0\r\n" +
		mediaLine + "\r\n" +
		"a=rtpmap:0 PCMU/8000\r\n" +
		"a=rtpmap:101 telephone-event/8000\r\n" +
		"a=sendrecv\r\n"
}
