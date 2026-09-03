// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"
	"testing"
)

func TestDefaultSDPConfigUsesCurrentCoreShape(t *testing.T) {
	config := DefaultSDPConfig("203.0.113.10", 40000)

	if config.SessionID != "0" {
		t.Fatalf("expected default session id 0, got %q", config.SessionID)
	}
	if config.LocalIP != "203.0.113.10" {
		t.Fatalf("expected local IP to be preserved, got %q", config.LocalIP)
	}
	if config.RTPPort != 40000 {
		t.Fatalf("expected RTP port to be preserved, got %d", config.RTPPort)
	}
	if config.PTime != 20 {
		t.Fatalf("expected default packet time 20ms, got %d", config.PTime)
	}
	if len(config.Codecs) != 2 {
		t.Fatalf("expected two default audio codecs, got %d", len(config.Codecs))
	}
	if config.Codecs[0] != CodecPCMU || config.Codecs[1] != CodecPCMA {
		t.Fatalf("unexpected codec order: %#v", config.Codecs)
	}
}

func TestSessionFacadeConvertsPhaseAndCodec(t *testing.T) {
	session, err := NewSession(context.Background(),
		WithSessionConfig(&Config{
			Server:            "sip.example.com",
			Port:              5060,
			Transport:         TransportUDP,
			RTPPortRangeStart: 30000,
			RTPPortRangeEnd:   30100,
		}),
		WithSessionDirection(CallDirectionInbound),
		WithSessionCallID("facade-call"),
		WithSessionCodec(&CodecPCMA),
	)
	if err != nil {
		t.Fatalf("expected session facade to create core-backed session: %v", err)
	}

	session.SetInboundSetupPhase(InboundSetupPhaseMediaFlowing)
	if phase := session.GetInboundSetupPhase(); phase != InboundSetupPhaseMediaFlowing {
		t.Fatalf("expected inbound phase round-trip, got %q", phase)
	}

	session.SetNegotiatedCodec(CodecPCMU.Name, int(CodecPCMU.ClockRate))
	codec := session.GetNegotiatedCodec()
	if codec == nil || *codec != CodecPCMU {
		t.Fatalf("expected negotiated codec round-trip, got %#v", codec)
	}

	info := session.GetInfo()
	if info.CallID != "facade-call" {
		t.Fatalf("expected call ID from core session, got %q", info.CallID)
	}
	if info.Direction != CallDirectionInbound {
		t.Fatalf("expected inbound direction, got %q", info.Direction)
	}
}

func TestSessionFacadeUsesCoreEventChannel(t *testing.T) {
	session, err := NewSession(context.Background(),
		WithSessionCallID("facade-events"),
		WithSessionConfig(&Config{
			Server:            "sip.example.com",
			Port:              5060,
			Transport:         TransportUDP,
			RTPPortRangeStart: 30000,
			RTPPortRangeEnd:   30100,
		}),
	)
	if err != nil {
		t.Fatalf("expected session facade to create core-backed session: %v", err)
	}

	if session.Events() != session.inner.Events() {
		t.Fatal("expected session facade to expose the core event channel")
	}
}
