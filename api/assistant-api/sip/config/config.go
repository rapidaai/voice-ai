// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import (
	assistant_config "github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

// Resolver owns conversion from app SIP config plus vault values into runtime config.
type Resolver struct {
	appConfig *assistant_config.SIPConfig
}

func NewResolver(appConfig *assistant_config.SIPConfig) Resolver {
	return Resolver{appConfig: appConfig}
}

func NewListenConfig(appConfig *assistant_config.SIPConfig) *ListenConfig {
	if appConfig == nil {
		return &ListenConfig{}
	}

	transportType := TransportUDP
	switch appConfig.Transport {
	case string(TransportTCP):
		transportType = TransportTCP
	case string(TransportTLS):
		transportType = TransportTLS
	}
	return &ListenConfig{
		Address:                 appConfig.Server,
		ExternalIP:              appConfig.ExternalIP,
		AllowLoopbackExternalIP: appConfig.AllowLoopbackExternalIP,
		Port:                    appConfig.Port,
		Transport:               transportType,
	}
}

func NewServerConfig(
	appConfig *assistant_config.SIPConfig,
	logger commons.Logger,
) *ServerConfig {
	if appConfig == nil {
		return &ServerConfig{
			ListenConfig: NewListenConfig(nil),
			Logger:       logger,
		}
	}

	return &ServerConfig{
		ListenConfig:         NewListenConfig(appConfig),
		Logger:               logger,
		RTPPortRangeStart:    appConfig.RTPPortRangeStart,
		RTPPortRangeEnd:      appConfig.RTPPortRangeEnd,
		SymmetricRTP:         appConfig.SymmetricRTP,
		IgnoreLocalAddrInSDP: appConfig.IgnoreLocalAddrInSDP,
		MaxConcurrentCalls:   appConfig.MaxConcurrentCalls,
		CallAdmissionCPS:     appConfig.CallAdmissionCPS,
		CallAdmissionBurst:   appConfig.CallAdmissionBurst,
	}
}

func (c Resolver) InstanceID() string {
	if c.appConfig == nil {
		return ""
	}
	return c.appConfig.InstanceID
}

func (c Resolver) RuntimeConfig(vaultCredential *protos.VaultCredential) (*Config, error) {
	runtimeConfig, err := ParseConfigFromVault(vaultCredential)
	if err != nil {
		return nil, err
	}

	if c.appConfig == nil {
		if runtimeConfig.Port <= 0 {
			runtimeConfig.Port = DefaultProviderPort
		}
		return runtimeConfig, nil
	}

	if runtimeConfig.Port <= 0 && c.appConfig.Port > 0 {
		runtimeConfig.Port = c.appConfig.Port
	}
	if runtimeConfig.Transport == "" && c.appConfig.Transport != "" {
		runtimeConfig.Transport = Transport(c.appConfig.Transport)
	}
	if runtimeConfig.RTPPortRangeStart <= 0 && c.appConfig.RTPPortRangeStart > 0 {
		runtimeConfig.RTPPortRangeStart = c.appConfig.RTPPortRangeStart
	}
	if runtimeConfig.RTPPortRangeEnd <= 0 && c.appConfig.RTPPortRangeEnd > 0 {
		runtimeConfig.RTPPortRangeEnd = c.appConfig.RTPPortRangeEnd
	}
	if runtimeConfig.RegisterTimeout <= 0 && c.appConfig.RegisterTimeout > 0 {
		runtimeConfig.RegisterTimeout = c.appConfig.RegisterTimeout
	}
	if runtimeConfig.InviteTimeout <= 0 && c.appConfig.InviteTimeout > 0 {
		runtimeConfig.InviteTimeout = c.appConfig.InviteTimeout
	}
	if runtimeConfig.SessionTimeout <= 0 && c.appConfig.SessionTimeout > 0 {
		runtimeConfig.SessionTimeout = c.appConfig.SessionTimeout
	}
	if runtimeConfig.MediaTimeoutInitial <= 0 && c.appConfig.MediaTimeoutInitial > 0 {
		runtimeConfig.MediaTimeoutInitial = c.appConfig.MediaTimeoutInitial
	}
	if runtimeConfig.MediaTimeout <= 0 && c.appConfig.MediaTimeout > 0 {
		runtimeConfig.MediaTimeout = c.appConfig.MediaTimeout
	}
	if runtimeConfig.InboundAnswerMode == "" && c.appConfig.Inbound.AnswerMode != "" {
		runtimeConfig.InboundAnswerMode = InboundAnswerMode(c.appConfig.Inbound.AnswerMode)
	}
	if runtimeConfig.InboundMinRingDuration <= 0 && c.appConfig.Inbound.MinRingDuration > 0 {
		runtimeConfig.InboundMinRingDuration = c.appConfig.Inbound.MinRingDuration
	}
	if runtimeConfig.InboundMaxRingDuration <= 0 && c.appConfig.Inbound.MaxRingDuration > 0 {
		runtimeConfig.InboundMaxRingDuration = c.appConfig.Inbound.MaxRingDuration
	}
	if runtimeConfig.InboundACKTimeout <= 0 && c.appConfig.Inbound.ACKTimeout > 0 {
		runtimeConfig.InboundACKTimeout = c.appConfig.Inbound.ACKTimeout
	}
	if runtimeConfig.Port <= 0 {
		runtimeConfig.Port = DefaultProviderPort
	}

	return runtimeConfig, nil
}
