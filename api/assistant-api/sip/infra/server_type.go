// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"

type ServerState = internal_core.ServerState

const (
	ServerStateCreated = internal_core.ServerStateCreated
	ServerStateRunning = internal_core.ServerStateRunning
	ServerStateStopped = internal_core.ServerStateStopped
)

type CallAddress = internal_core.CallAddress

type SIPRequestContext = internal_core.SIPRequestContext

type SIPRequestIdentity struct {
	RequestURI  string
	CallAddress CallAddress
	// Deprecated: derived from CallAddress.FromURI. Do not write.
	FromIdentity string
	// Deprecated: derived from CallAddress.ToURI. Do not write.
	ToIdentity string
}

type Middleware = internal_core.Middleware

type Server struct {
	inner *internal_core.Server
}

type ListenConfig = internal_core.ListenConfig

type ServerConfig = internal_core.ServerConfig

func cloneConfig(config *Config) *internal_core.Config {
	if config == nil {
		return nil
	}
	cloned := *config
	return &cloned
}
