// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import (
	"fmt"

	"github.com/emiago/sipgo/sip"
	"github.com/rapidaai/pkg/commons"
)

// ListenConfig holds shared server configuration, not tenant-specific config.
type ListenConfig struct {
	Address                 string    `json:"address" mapstructure:"address"`
	ExternalIP              string    `json:"external_ip" mapstructure:"external_ip"`
	AllowLoopbackExternalIP bool      `json:"allow_loopback_external_ip" mapstructure:"allow_loopback_external_ip"`
	Port                    int       `json:"port" mapstructure:"port"`
	Transport               Transport `json:"transport" mapstructure:"transport"`
}

// GetExternalIP returns the advertised IP for SDP and SIP Contact headers.
func (c *ListenConfig) GetExternalIP() string {
	if c == nil {
		return ""
	}
	if c.ExternalIP != "" {
		return c.ExternalIP
	}
	return c.Address
}

// GetBindAddress returns the local interface address used for RTP sockets.
func (c *ListenConfig) GetBindAddress() string {
	if c == nil {
		return ""
	}
	return c.Address
}

func (c *ListenConfig) GetListenAddr() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

func (c *ListenConfig) SIPContactHeader() sip.ContactHeader {
	contactURI := sip.Uri{
		Scheme: "sip",
		Host:   c.GetExternalIP(),
		Port:   c.Port,
	}
	if c.Transport == TransportTLS {
		contactURI.Scheme = "sips"
	}
	if c.Transport == TransportTCP || c.Transport == TransportTLS {
		contactURI.UriParams = sip.NewParams()
		contactURI.UriParams.Add("transport", string(c.Transport))
	}
	return sip.ContactHeader{Address: contactURI}
}

// ServerConfig holds shared server configuration. Tenant config resolves per call.
type ServerConfig struct {
	ListenConfig         *ListenConfig
	Logger               commons.Logger
	RTPPortRangeStart    int
	RTPPortRangeEnd      int
	SymmetricRTP         bool
	IgnoreLocalAddrInSDP bool
	MaxConcurrentCalls   int
	CallAdmissionCPS     int
	CallAdmissionBurst   int
}

func (c *ServerConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("server config is required")
	}
	if c.ListenConfig == nil {
		return fmt.Errorf("listen config is required")
	}
	if c.ListenConfig.Address == "" {
		return fmt.Errorf("listen address is required")
	}
	if c.ListenConfig.Port <= 0 || c.ListenConfig.Port > maxPort {
		return fmt.Errorf("invalid listen port: %d", c.ListenConfig.Port)
	}
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	if c.RTPPortRangeStart <= 0 || c.RTPPortRangeEnd <= 0 {
		return fmt.Errorf("rtp_port_range must be specified")
	}
	if c.RTPPortRangeStart > c.RTPPortRangeEnd {
		return fmt.Errorf("rtp_port_range_start must be less than or equal to rtp_port_range_end")
	}
	if c.MaxConcurrentCalls < 0 {
		return fmt.Errorf("max_concurrent_calls must be greater than or equal to zero")
	}
	if c.CallAdmissionCPS < 0 {
		return fmt.Errorf("call_admission_cps must be greater than or equal to zero")
	}
	if c.CallAdmissionBurst < 0 {
		return fmt.Errorf("call_admission_burst must be greater than or equal to zero")
	}
	if c.CallAdmissionCPS > 0 && c.CallAdmissionBurst == 0 {
		return fmt.Errorf("call_admission_burst is required when call_admission_cps is set")
	}
	if c.CallAdmissionBurst > 0 && c.CallAdmissionCPS == 0 {
		return fmt.Errorf("call_admission_cps is required when call_admission_burst is set")
	}
	return nil
}
