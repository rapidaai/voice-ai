// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"errors"
	"testing"
	"time"

	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
)

func TestConfigTypeConversionsPreserveFields(t *testing.T) {
	config := &Config{
		Server:              "sip.example.com",
		Username:            "auth-user",
		Password:            "auth-pass",
		Realm:               "sip.example.com",
		Domain:              "voice.example.com",
		CallerID:            "+15550001111",
		CustomHeaders:       map[string]string{"X-Rapida-Test": "ok"},
		Port:                5061,
		Transport:           TransportTLS,
		RTPPortRangeStart:   30000,
		RTPPortRangeEnd:     30100,
		SRTPEnabled:         true,
		RegisterTimeout:     11 * time.Second,
		InviteTimeout:       12 * time.Second,
		SessionTimeout:      13 * time.Second,
		MediaTimeoutInitial: 14 * time.Second,
		MediaTimeout:        15 * time.Second,
		KeepAliveEnabled:    true,
	}

	coreConfig := config.toCore()
	if coreConfig.Server != config.Server || coreConfig.Username != config.Username || coreConfig.Password != config.Password {
		t.Fatalf("core config did not preserve credentials: %#v", coreConfig)
	}
	if coreConfig.Transport != internal_core.TransportTLS || !coreConfig.SRTPEnabled || !coreConfig.KeepAliveEnabled {
		t.Fatalf("core config did not preserve transport flags: %#v", coreConfig)
	}
	if coreConfig.CustomHeaders["X-Rapida-Test"] != "ok" {
		t.Fatalf("core config did not preserve custom headers: %#v", coreConfig.CustomHeaders)
	}

	roundTripConfig := configFromCore(coreConfig)
	if roundTripConfig.Server != config.Server || roundTripConfig.Transport != config.Transport {
		t.Fatalf("round-trip config mismatch: %#v", roundTripConfig)
	}
	if roundTripConfig.RegisterTimeout != config.RegisterTimeout ||
		roundTripConfig.InviteTimeout != config.InviteTimeout ||
		roundTripConfig.SessionTimeout != config.SessionTimeout ||
		roundTripConfig.MediaTimeoutInitial != config.MediaTimeoutInitial ||
		roundTripConfig.MediaTimeout != config.MediaTimeout {
		t.Fatalf("round-trip timeout mismatch: %#v", roundTripConfig)
	}
}

func TestConfigDefaultsAndValidation(t *testing.T) {
	var nilConfig *Config
	nilConfig.ApplyOperationalDefaults(5060, TransportUDP, 30000, 30100)
	nilConfig.ApplyTimeoutDefaults(time.Second, 2*time.Second, 3*time.Second)
	nilConfig.ApplyMediaTimeoutDefaults(7*time.Second, 8*time.Second)
	nilConfig.ApplyInboundAnswerDefaults(InboundAnswerModeAfterMinRingDuration, time.Second, 2*time.Second, 3*time.Second)

	config := &Config{Server: "sip.example.com"}
	config.ApplyOperationalDefaults(5060, TransportUDP, 30000, 30100)
	config.ApplyTimeoutDefaults(time.Second, 2*time.Second, 3*time.Second)
	config.ApplyMediaTimeoutDefaults(7*time.Second, 8*time.Second)
	config.ApplyInboundAnswerDefaults(InboundAnswerModeAfterMinRingDuration, 4*time.Second, 5*time.Second, 6*time.Second)

	if config.Port != 5060 || config.Transport != TransportUDP {
		t.Fatalf("operational defaults were not applied: %#v", config)
	}
	if config.RTPPortRangeStart != 30000 || config.RTPPortRangeEnd != 30100 {
		t.Fatalf("RTP defaults were not applied: %#v", config)
	}
	if config.RegisterTimeout != time.Second || config.InviteTimeout != 2*time.Second || config.SessionTimeout != 3*time.Second {
		t.Fatalf("timeout defaults were not applied: %#v", config)
	}
	if config.MediaTimeoutInitial != 7*time.Second || config.MediaTimeout != 8*time.Second {
		t.Fatalf("media timeout defaults were not applied: %#v", config)
	}
	if config.InboundAnswerMode != InboundAnswerModeAfterMinRingDuration ||
		config.InboundMinRingDuration != 4*time.Second ||
		config.InboundMaxRingDuration != 5*time.Second ||
		config.InboundACKTimeout != 6*time.Second {
		t.Fatalf("inbound answer defaults were not applied: %#v", config)
	}
	if err := config.ValidateRTP(); err != nil {
		t.Fatalf("expected valid RTP config, got %v", err)
	}
}

func TestSIPErrorAndTransportHelpers(t *testing.T) {
	rootErr := errors.New("network down")
	err := NewSIPError("invite", "call-123", "send failed", rootErr)

	if !errors.Is(err, rootErr) {
		t.Fatal("expected SIPError to unwrap the root error")
	}
	if got := err.Error(); got != "sip invite [call_id=call-123]: send failed: network down" {
		t.Fatalf("unexpected SIPError string: %q", got)
	}

	if !TransportUDP.IsValid() || !TransportTCP.IsValid() || !TransportTLS.IsValid() {
		t.Fatal("expected known transports to be valid")
	}
	if Transport("ws").IsValid() {
		t.Fatal("expected unknown transport to be invalid")
	}
	if TransportTLS.String() != "tls" {
		t.Fatalf("expected TLS transport string tls, got %q", TransportTLS.String())
	}
}

func TestCallStateHelpers(t *testing.T) {
	for _, state := range []CallState{CallStateEnded, CallStateFailed, CallStateCancelled} {
		if !state.IsTerminal() {
			t.Fatalf("expected %s to be terminal", state)
		}
	}
	for _, state := range []CallState{CallStateConnected, CallStateRinging, CallStateOnHold, CallStateTransferring, CallStateBridgeConnected} {
		if !state.IsActive() {
			t.Fatalf("expected %s to be active", state)
		}
	}
	if CallStateInitializing.IsTerminal() || CallStateInitializing.IsActive() {
		t.Fatal("initializing should not be active or terminal")
	}
	if callStateFromCore(internal_core.CallStateConnected) != CallStateConnected {
		t.Fatal("expected core connected state to convert to infra connected state")
	}
}

func TestSessionInfoDuration(t *testing.T) {
	connectedAt := time.Now().Add(-2 * time.Second)
	endedAt := connectedAt.Add(1500 * time.Millisecond)

	info := &SessionInfo{ConnectedTime: &connectedAt, EndTime: &endedAt}
	if info.GetDuration() != 1500*time.Millisecond {
		t.Fatalf("expected completed duration 1.5s, got %s", info.GetDuration())
	}

	activeInfo := &SessionInfo{ConnectedTime: &connectedAt}
	if activeInfo.GetDuration() <= 0 {
		t.Fatal("expected active session duration to be positive")
	}

	if (&SessionInfo{}).GetDuration() != 0 {
		t.Fatal("expected unconnected session duration to be zero")
	}
}

func TestDisconnectMetadataAndEventConversions(t *testing.T) {
	metadata := DisconnectMetadata{
		Reason:             DisconnectReasonBusy,
		Text:               "Busy Here",
		Raw:                "SIP ;cause=486 ;text=\"Busy Here\"",
		ProviderStatusCode: 486,
	}
	coreMetadata := metadata.toCore()
	if coreMetadata.Reason != metadata.Reason || coreMetadata.ProviderStatusCode != metadata.ProviderStatusCode {
		t.Fatalf("core disconnect metadata mismatch: %#v", coreMetadata)
	}
	if disconnectMetadataFromCore(coreMetadata) != metadata {
		t.Fatalf("disconnect metadata round-trip mismatch: %#v", disconnectMetadataFromCore(coreMetadata))
	}

	event := NewEvent(EventTypeConnected, "call-123", map[string]interface{}{"state": "connected"})
	if event.Type != EventTypeConnected || event.CallID != "call-123" || event.Timestamp.IsZero() {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func TestSDPTypeHelpers(t *testing.T) {
	if nilInfo := (*SDPMediaInfo)(nil); nilInfo.IsHold() {
		t.Fatal("nil SDP media info should not be hold")
	}
	for _, mediaInfo := range []*SDPMediaInfo{
		{Direction: SDPDirectionSendOnly},
		{Direction: SDPDirectionInactive},
		{ConnectionIP: "0.0.0.0", Direction: SDPDirectionSendRecv},
	} {
		if !mediaInfo.IsHold() {
			t.Fatalf("expected SDP media info to be hold: %#v", mediaInfo)
		}
	}

	if codec := GetCodecByPayloadType(0); codec == nil || *codec != CodecPCMU {
		t.Fatalf("expected payload type 0 to resolve to PCMU, got %#v", codec)
	}
	if codec := GetCodecByName("PCMA"); codec == nil || *codec != CodecPCMA {
		t.Fatalf("expected PCMA lookup, got %#v", codec)
	}

	coreInfo := &internal_core.SDPMediaInfo{
		ConnectionIP:   "203.0.113.10",
		AudioPort:      40000,
		PayloadTypes:   []uint8{0, 8},
		PreferredCodec: &internal_core.CodecPCMA,
		Direction:      internal_core.SDPDirectionSendRecv,
	}
	mediaInfo := sdpInfoFromCore(coreInfo)
	coreInfo.PayloadTypes[0] = 99
	if mediaInfo.PayloadTypes[0] != 0 {
		t.Fatalf("expected SDP payload types to be copied, got %#v", mediaInfo.PayloadTypes)
	}
	if mediaInfo.PreferredCodec == nil || *mediaInfo.PreferredCodec != CodecPCMA {
		t.Fatalf("expected preferred codec PCMA, got %#v", mediaInfo.PreferredCodec)
	}
}

func TestListenConfigAndServerConfigConversions(t *testing.T) {
	listenConfig := &ListenConfig{
		Address:                 "127.0.0.1",
		ExternalIP:              "203.0.113.10",
		AllowLoopbackExternalIP: true,
		Port:                    5060,
		Transport:               TransportUDP,
	}

	if listenConfig.GetExternalIP() != "203.0.113.10" {
		t.Fatalf("expected external IP override, got %q", listenConfig.GetExternalIP())
	}
	if listenConfig.GetBindAddress() != "127.0.0.1" || listenConfig.GetListenAddr() != "127.0.0.1:5060" {
		t.Fatalf("unexpected listen config addresses: bind=%q listen=%q", listenConfig.GetBindAddress(), listenConfig.GetListenAddr())
	}

	coreListenConfig := listenConfig.toCore()
	roundTripListenConfig := listenConfigFromCore(coreListenConfig)
	if roundTripListenConfig.Address != listenConfig.Address ||
		roundTripListenConfig.ExternalIP != listenConfig.ExternalIP ||
		roundTripListenConfig.Transport != listenConfig.Transport {
		t.Fatalf("listen config round-trip mismatch: %#v", roundTripListenConfig)
	}

	serverConfig := (&ServerConfig{
		ListenConfig:      listenConfig,
		RTPPortRangeStart: 30000,
		RTPPortRangeEnd:   30100,
	}).toCore()
	if serverConfig.ListenConfig.Address != listenConfig.Address || serverConfig.RTPPortRangeStart != 30000 {
		t.Fatalf("server config conversion mismatch: %#v", serverConfig)
	}
}

func TestSIPRequestContextMiddleware(t *testing.T) {
	requestContext := &SIPRequestContext{}

	var order []string
	middlewares := []Middleware{
		func(ctx *SIPRequestContext) error {
			order = append(order, "first")
			return nil
		},
		func(ctx *SIPRequestContext) error {
			order = append(order, "second")
			return nil
		},
		func(ctx *SIPRequestContext) error {
			order = append(order, "final")
			ctx.Config = &Config{Server: "sip.example.com"}
			return nil
		},
	}

	for _, middleware := range middlewares {
		if err := middleware(requestContext); err != nil {
			t.Fatalf("middleware returned error: %v", err)
		}
	}
	if requestContext.Config == nil || requestContext.Config.Server != "sip.example.com" {
		t.Fatalf("unexpected middleware context: %#v", requestContext)
	}
	expectedOrder := []string{"first", "second", "final"}
	for i := range expectedOrder {
		if order[i] != expectedOrder[i] {
			t.Fatalf("middleware order mismatch at %d: got %q want %q", i, order[i], expectedOrder[i])
		}
	}
}

func TestRegistrationTypeConversionAndErrors(t *testing.T) {
	if ErrMissingDID != internal_core.ErrMissingDID || ErrAuthFailed != internal_core.ErrAuthFailed {
		t.Fatal("registration errors should expose core registration errors")
	}

	registration := &Registration{
		DID: "+15551234567",
		Config: &Config{
			Server:    "sip.example.com",
			Username:  "auth-user",
			Password:  "auth-pass",
			Transport: TransportUDP,
			Port:      5060,
		},
		AssistantID: 77,
		ExpiresIn:   120,
	}
	coreRegistration := registration.toCore()
	if coreRegistration.DID != registration.DID ||
		coreRegistration.AssistantID != registration.AssistantID ||
		coreRegistration.ExpiresIn != registration.ExpiresIn {
		t.Fatalf("registration conversion mismatch: %#v", coreRegistration)
	}
	if err := registration.Validate(); err != nil {
		t.Fatalf("expected registration to validate, got %v", err)
	}
}

func TestRTPTypeManualHandlerFallback(t *testing.T) {
	handler := &RTPHandler{}
	handler.SetCodec(&CodecPCMA)

	if !handler.IsRunning() {
		t.Fatal("manual RTP handler fallback should report running")
	}
	if codec := handler.GetCodec(); codec == nil || *codec != CodecPCMA {
		t.Fatalf("expected manual RTP codec PCMA, got %#v", codec)
	}
	if handler.AudioIn() == nil {
		t.Fatal("expected manual RTP input channel")
	}
	if err := handler.EnqueueAudio([]byte{0x01}); err != nil {
		t.Fatalf("expected manual RTP output enqueue, got %v", err)
	}

	handler.FlushAudioOut()
	if handler.flushAudioCh == nil {
		t.Fatal("expected flush channel to be initialized")
	}
	if err := handler.Stop(); err != nil {
		t.Fatalf("manual RTP stop should not fail: %v", err)
	}
}

func TestLifecycleAndHealthTypeConversions(t *testing.T) {
	if LifecycleReasonOutboundNoAnswer.String() != "outbound_no_answer" {
		t.Fatalf("unexpected lifecycle reason string: %q", LifecycleReasonOutboundNoAnswer.String())
	}
	if LifecycleReasonOutboundNoAnswer.toCore() != internal_core.LifecycleReasonOutboundNoAnswer {
		t.Fatal("lifecycle reason did not convert to core")
	}

	snapshot := serverHealthSnapshotFromCore(internal_core.ServerHealthSnapshot{
		Ready:                   true,
		Reason:                  "ready",
		State:                   internal_core.ServerStateRunning,
		ActiveCalls:             2,
		RTPPortsInUse:           3,
		RTPPortBindAttempts:     7,
		RTPPortBindFailures:     1,
		RTPPortRangeExhaustions: 1,
	})
	if !snapshot.Ready || snapshot.State != ServerStateRunning || snapshot.ActiveCalls != 2 || snapshot.RTPPortsInUse != 3 ||
		snapshot.RTPPortBindAttempts != 7 || snapshot.RTPPortBindFailures != 1 || snapshot.RTPPortRangeExhaustions != 1 {
		t.Fatalf("health snapshot conversion mismatch: %#v", snapshot)
	}
}

func TestOutboundTypeConversions(t *testing.T) {
	headers := map[string]string{"X-Test": "ok"}
	outboundConfig := OutboundConfig{
		Mode:                OutboundModeTrunkTermination,
		Address:             "sip.example.com",
		Port:                5060,
		Transport:           TransportUDP,
		Domain:              "voice.example.com",
		Auth:                SIPAuthConfig{Username: "auth-user", Password: "auth-pass", Realm: "sip.example.com"},
		Headers:             headers,
		RingingTimeout:      5 * time.Second,
		MaxCallDuration:     time.Minute,
		MediaTimeoutInitial: 20 * time.Second,
		MediaTimeout:        10 * time.Second,
	}
	coreOutboundConfig := outboundConfig.toCore()
	headers["X-Test"] = "mutated"

	if coreOutboundConfig.Headers["X-Test"] != "ok" {
		t.Fatalf("expected outbound headers to be copied, got %#v", coreOutboundConfig.Headers)
	}
	if coreOutboundConfig.Mode != internal_core.OutboundModeTrunkTermination ||
		coreOutboundConfig.Auth.Username != outboundConfig.Auth.Username ||
		coreOutboundConfig.MediaTimeoutInitial != outboundConfig.MediaTimeoutInitial ||
		coreOutboundConfig.MediaTimeout != outboundConfig.MediaTimeout {
		t.Fatalf("outbound config conversion mismatch: %#v", coreOutboundConfig)
	}

	request := OutboundInviteRequest{
		Config: outboundConfig,
		Identity: OutboundCallIdentity{
			ToUser:   "+15550001111",
			FromUser: "+15550002222",
		},
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("expected outbound invite request to validate, got %v", err)
	}
	if request.toCore().Identity.FromUser != "+15550002222" {
		t.Fatalf("outbound invite identity conversion mismatch: %#v", request.toCore().Identity)
	}
}
