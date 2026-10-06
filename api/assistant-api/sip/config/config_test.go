// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import (
	"testing"
	"time"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRuntimeConfigAppliesAppDefaults(t *testing.T) {
	vaultValue, err := structpb.NewStruct(map[string]interface{}{
		"sip_server":   "pbx.example.com",
		"sip_username": "user",
		"sip_password": "pass",
	})
	if err != nil {
		t.Fatalf("failed to create vault value: %v", err)
	}

	runtimeConfig, err := NewResolver(&assistant_config.SIPConfig{
		Port:                5070,
		Transport:           "tcp",
		RTPPortRangeStart:   10000,
		RTPPortRangeEnd:     10100,
		RegisterTimeout:     5 * time.Second,
		InviteTimeout:       30 * time.Second,
		SessionTimeout:      45 * time.Minute,
		MediaTimeoutInitial: 20 * time.Second,
		MediaTimeout:        10 * time.Second,
		Inbound: assistant_config.SIPInboundConfig{
			AnswerMode:      string(InboundAnswerModeAfterMinRingDuration),
			MinRingDuration: 50 * time.Millisecond,
			MaxRingDuration: 5 * time.Second,
			ACKTimeout:      2 * time.Second,
		},
	}).RuntimeConfig(&protos.VaultCredential{Value: vaultValue})
	if err != nil {
		t.Fatalf("RuntimeConfig() error = %v", err)
	}

	if runtimeConfig.Port != 5070 {
		t.Fatalf("Port = %d, want 5070", runtimeConfig.Port)
	}
	if runtimeConfig.Transport != TransportTCP {
		t.Fatalf("Transport = %q, want tcp", runtimeConfig.Transport)
	}
	if runtimeConfig.RTPPortRangeStart != 10000 || runtimeConfig.RTPPortRangeEnd != 10100 {
		t.Fatalf("RTP range = %d-%d, want 10000-10100", runtimeConfig.RTPPortRangeStart, runtimeConfig.RTPPortRangeEnd)
	}
	if runtimeConfig.RegisterTimeout != 5*time.Second ||
		runtimeConfig.InviteTimeout != 30*time.Second ||
		runtimeConfig.SessionTimeout != 45*time.Minute ||
		runtimeConfig.MediaTimeoutInitial != 20*time.Second ||
		runtimeConfig.MediaTimeout != 10*time.Second {
		t.Fatalf("runtime timeout defaults not applied: %#v", runtimeConfig)
	}
	if runtimeConfig.InboundAnswerMode != InboundAnswerModeAfterMinRingDuration ||
		runtimeConfig.InboundMinRingDuration != 50*time.Millisecond ||
		runtimeConfig.InboundMaxRingDuration != 5*time.Second ||
		runtimeConfig.InboundACKTimeout != 2*time.Second {
		t.Fatalf("inbound answer defaults not applied: %#v", runtimeConfig)
	}
}

func TestRuntimeConfigPreservesVaultValues(t *testing.T) {
	vaultValue, err := structpb.NewStruct(map[string]interface{}{
		"sip_server": "pbx.example.com",
		"sip_port":   5099,
	})
	if err != nil {
		t.Fatalf("failed to create vault value: %v", err)
	}

	runtimeConfig, err := NewResolver(&assistant_config.SIPConfig{Port: 5070}).RuntimeConfig(&protos.VaultCredential{Value: vaultValue})
	if err != nil {
		t.Fatalf("RuntimeConfig() error = %v", err)
	}
	if runtimeConfig.Port != 5099 {
		t.Fatalf("Port = %d, want 5099", runtimeConfig.Port)
	}
}

func TestRuntimeConfigDefaultsPortWithoutAppConfig(t *testing.T) {
	vaultValue, err := structpb.NewStruct(map[string]interface{}{
		"sip_server": "pbx.example.com",
	})
	if err != nil {
		t.Fatalf("failed to create vault value: %v", err)
	}

	runtimeConfig, err := NewResolver(nil).RuntimeConfig(&protos.VaultCredential{Value: vaultValue})
	if err != nil {
		t.Fatalf("RuntimeConfig() error = %v", err)
	}
	if runtimeConfig.Port != DefaultProviderPort {
		t.Fatalf("Port = %d, want %d", runtimeConfig.Port, DefaultProviderPort)
	}
}

func TestConfigInstanceID(t *testing.T) {
	if got := NewResolver(&assistant_config.SIPConfig{InstanceID: "sip-01"}).InstanceID(); got != "sip-01" {
		t.Fatalf("InstanceID() = %q, want sip-01", got)
	}
	if got := NewResolver(nil).InstanceID(); got != "" {
		t.Fatalf("InstanceID() = %q, want empty", got)
	}
}

func TestNewListenConfigMapsTransportAndAddresses(t *testing.T) {
	listenConfig := NewListenConfig(&assistant_config.SIPConfig{
		Server:                  "0.0.0.0",
		ExternalIP:              "203.0.113.10",
		AllowLoopbackExternalIP: true,
		Port:                    5070,
		Transport:               "tcp",
	})

	if listenConfig.Address != "0.0.0.0" {
		t.Fatalf("Address = %q, want 0.0.0.0", listenConfig.Address)
	}
	if listenConfig.ExternalIP != "203.0.113.10" {
		t.Fatalf("ExternalIP = %q, want 203.0.113.10", listenConfig.ExternalIP)
	}
	if !listenConfig.AllowLoopbackExternalIP {
		t.Fatal("AllowLoopbackExternalIP = false, want true")
	}
	if listenConfig.Port != 5070 {
		t.Fatalf("Port = %d, want 5070", listenConfig.Port)
	}
	if listenConfig.Transport != TransportTCP {
		t.Fatalf("Transport = %q, want tcp", listenConfig.Transport)
	}
}

func TestNewListenConfigDefaultsTransportToUDP(t *testing.T) {
	listenConfig := NewListenConfig(&assistant_config.SIPConfig{Transport: "invalid"})

	if listenConfig.Transport != TransportUDP {
		t.Fatalf("Transport = %q, want udp", listenConfig.Transport)
	}
}

func TestNewServerConfigMapsSharedRuntimeConfig(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	serverConfig := NewServerConfig(
		&assistant_config.SIPConfig{
			Server:               "0.0.0.0",
			Port:                 5070,
			Transport:            "tls",
			RTPPortRangeStart:    10000,
			RTPPortRangeEnd:      10100,
			SymmetricRTP:         true,
			IgnoreLocalAddrInSDP: true,
			MaxConcurrentCalls:   12,
			CallAdmissionCPS:     3,
			CallAdmissionBurst:   6,
		},
		logger,
	)

	if serverConfig.ListenConfig.Transport != TransportTLS {
		t.Fatalf("Transport = %q, want tls", serverConfig.ListenConfig.Transport)
	}
	if serverConfig.RTPPortRangeStart != 10000 || serverConfig.RTPPortRangeEnd != 10100 {
		t.Fatalf("RTP range = %d-%d, want 10000-10100", serverConfig.RTPPortRangeStart, serverConfig.RTPPortRangeEnd)
	}
	if !serverConfig.SymmetricRTP || !serverConfig.IgnoreLocalAddrInSDP {
		t.Fatalf("server RTP booleans not mapped: %#v", serverConfig)
	}
	if serverConfig.MaxConcurrentCalls != 12 ||
		serverConfig.CallAdmissionCPS != 3 ||
		serverConfig.CallAdmissionBurst != 6 {
		t.Fatalf("admission config not mapped: %#v", serverConfig)
	}
	if serverConfig.Logger == nil {
		t.Fatal("Logger is nil")
	}
}

func TestNewServerConfigNilAppConfigStillReturnsValidationObject(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	serverConfig := NewServerConfig(nil, logger)

	if serverConfig == nil || serverConfig.ListenConfig == nil {
		t.Fatal("expected non-nil server config and listen config")
	}
	if err := serverConfig.Validate(); err == nil {
		t.Fatal("expected nil app config server validation to fail")
	}
}
