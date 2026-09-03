// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"net/netip"
	"strings"
)

type ServerHealthSnapshot struct {
	Ready                   bool
	Reason                  string
	State                   ServerState
	ActiveCalls             int
	RTPPortsInUse           int
	RTPPortBindAttempts     uint64
	RTPPortBindFailures     uint64
	RTPPortRangeExhaustions uint64
}

func (s *Server) HealthSnapshot() ServerHealthSnapshot {
	if s == nil {
		return ServerHealthSnapshot{Reason: "server_nil"}
	}

	state := ServerState(s.state.Load())
	snapshot := ServerHealthSnapshot{
		State:       state,
		ActiveCalls: s.SessionCount(),
	}
	if s.rtpPortStats != nil {
		snapshot.RTPPortsInUse = int(s.rtpPortStats.portsInUse.Load())
		snapshot.RTPPortBindAttempts = s.rtpPortStats.bindAttempts.Load()
		snapshot.RTPPortBindFailures = s.rtpPortStats.bindFailures.Load()
		snapshot.RTPPortRangeExhaustions = s.rtpPortStats.rangeExhaustions.Load()
	}

	if state != ServerStateRunning {
		snapshot.Reason = "server_not_running"
		return snapshot
	}
	if s.client == nil {
		snapshot.Reason = "sip_client_unavailable"
		return snapshot
	}
	if s.server == nil {
		snapshot.Reason = "sip_listener_unavailable"
		return snapshot
	}
	if s.listenConfig == nil {
		snapshot.Reason = "listen_config_unavailable"
		return snapshot
	}
	if s.dialogClientCache == nil {
		snapshot.Reason = "outbound_dialog_cache_unavailable"
		return snapshot
	}
	if reason := outboundAdvertisedAddressHealthReason(s.listenConfig); reason != "" {
		snapshot.Reason = reason
		return snapshot
	}

	snapshot.Ready = true
	snapshot.Reason = "ready"
	return snapshot
}

func outboundAdvertisedAddressHealthReason(listenConfig *ListenConfig) string {
	if listenConfig == nil {
		return "listen_config_unavailable"
	}
	externalIP := strings.TrimSpace(listenConfig.ExternalIP)
	if externalIP == "" {
		return "external_ip_not_configured"
	}
	advertisedIP, err := netip.ParseAddr(externalIP)
	if err != nil {
		return "external_ip_invalid"
	}
	if advertisedIP.IsUnspecified() {
		return "external_ip_unspecified"
	}
	if advertisedIP.IsLoopback() && !listenConfig.AllowLoopbackExternalIP {
		return "external_ip_loopback"
	}
	return ""
}
