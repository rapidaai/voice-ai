// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/emiago/sipgo/sip"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type Transport string

const (
	TransportUDP Transport = "udp"
	TransportTCP Transport = "tcp"
	TransportTLS Transport = "tls"
)

func (t Transport) String() string {
	return string(t)
}

func (t Transport) IsValid() bool {
	switch t {
	case TransportUDP, TransportTCP, TransportTLS:
		return true
	default:
		return false
	}
}

type InboundAnswerMode string

const (
	InboundAnswerModeImmediate            InboundAnswerMode = "answer_immediately"
	InboundAnswerModeAfterMinRingDuration InboundAnswerMode = "answer_after_min_ring_ms"
)

func (m InboundAnswerMode) IsValid() bool {
	switch m {
	case "", InboundAnswerModeImmediate, InboundAnswerModeAfterMinRingDuration:
		return true
	default:
		return false
	}
}

// Config combines provider SIP settings from vault with platform runtime settings.
type Config struct {
	Server   string `json:"sip_server" mapstructure:"sip_server"`
	Username string `json:"sip_username" mapstructure:"sip_username"`
	Password string `json:"sip_password" mapstructure:"sip_password"`
	Realm    string `json:"sip_realm" mapstructure:"sip_realm"`
	Domain   string `json:"sip_domain,omitempty" mapstructure:"sip_domain"`

	// CallerID overrides the From header user in outbound calls.
	CallerID string `json:"sip_caller_id,omitempty" mapstructure:"sip_caller_id"`

	// CustomHeaders are added to outbound INVITE requests.
	CustomHeaders map[string]string `json:"sip_headers,omitempty" mapstructure:"sip_headers"`

	Port              int       `json:"sip_port" mapstructure:"sip_port"`
	Transport         Transport `json:"sip_transport" mapstructure:"sip_transport"`
	RTPPortRangeStart int       `json:"rtp_port_range_start" mapstructure:"rtp_port_range_start"`
	RTPPortRangeEnd   int       `json:"rtp_port_range_end" mapstructure:"rtp_port_range_end"`
	SRTPEnabled       bool      `json:"srtp_enabled" mapstructure:"srtp_enabled"`

	RegisterTimeout     time.Duration `json:"register_timeout,omitempty" mapstructure:"register_timeout"`
	InviteTimeout       time.Duration `json:"invite_timeout,omitempty" mapstructure:"invite_timeout"`
	SessionTimeout      time.Duration `json:"session_timeout,omitempty" mapstructure:"session_timeout"`
	MediaTimeoutInitial time.Duration `json:"media_timeout_initial,omitempty" mapstructure:"media_timeout_initial"`
	MediaTimeout        time.Duration `json:"media_timeout,omitempty" mapstructure:"media_timeout"`
	KeepAliveEnabled    bool          `json:"keepalive_enabled,omitempty" mapstructure:"keepalive_enabled"`

	InboundAnswerMode      InboundAnswerMode `json:"inbound_answer_mode,omitempty" mapstructure:"inbound_answer_mode"`
	InboundMinRingDuration time.Duration     `json:"inbound_min_ring_duration,omitempty" mapstructure:"inbound_min_ring_duration"`
	InboundMaxRingDuration time.Duration     `json:"inbound_max_ring_duration,omitempty" mapstructure:"inbound_max_ring_duration"`
	InboundACKTimeout      time.Duration     `json:"inbound_ack_timeout,omitempty" mapstructure:"inbound_ack_timeout"`
}

func (c *Config) Validate() error {
	return c.ValidateRTP()
}

func (c *Config) EffectiveRegisterTimeout() time.Duration {
	if c != nil && c.RegisterTimeout > 0 {
		return c.RegisterTimeout
	}
	return defaultRegisterTimeout
}

type InboundAnswerPolicy struct {
	Mode            InboundAnswerMode
	MinRingDuration time.Duration
	ACKTimeout      time.Duration
}

func DefaultInboundAnswerPolicy() InboundAnswerPolicy {
	return InboundAnswerPolicy{
		Mode:       InboundAnswerModeImmediate,
		ACKTimeout: defaultInboundACKTime,
	}
}

func (c *Config) EffectiveInboundAnswerPolicy(defaultACKTimeout time.Duration) InboundAnswerPolicy {
	policy := DefaultInboundAnswerPolicy()
	if defaultACKTimeout > 0 {
		policy.ACKTimeout = defaultACKTimeout
	}
	if c == nil {
		return policy
	}
	if c.InboundAnswerMode != "" {
		policy.Mode = c.InboundAnswerMode
	}
	if c.InboundMinRingDuration > 0 {
		policy.MinRingDuration = c.InboundMinRingDuration
	}
	if c.InboundACKTimeout > 0 {
		policy.ACKTimeout = c.InboundACKTimeout
	}
	return policy
}

func (c *Config) ValidateRTP() error {
	if c.Server == "" {
		return fmt.Errorf("%w: sip_server is required", ErrInvalidConfig)
	}
	if c.Port <= 0 || c.Port > maxPort {
		return fmt.Errorf("%w: sip_port must be between 1 and 65535", ErrInvalidConfig)
	}
	if c.RTPPortRangeStart <= 0 || c.RTPPortRangeEnd <= 0 {
		return fmt.Errorf("%w: rtp_port_range must be specified", ErrInvalidConfig)
	}
	if c.RTPPortRangeStart > c.RTPPortRangeEnd {
		return fmt.Errorf("%w: rtp_port_range_start must be less than or equal to rtp_port_range_end", ErrInvalidConfig)
	}
	if c.RTPPortRangeStart < 1024 {
		return fmt.Errorf("%w: rtp_port_range_start must be >= 1024 (non-privileged port)", ErrInvalidConfig)
	}
	if !c.Transport.IsValid() && c.Transport != "" {
		return fmt.Errorf("%w: invalid transport: %s", ErrInvalidConfig, c.Transport)
	}
	if !c.InboundAnswerMode.IsValid() {
		return fmt.Errorf("%w: invalid inbound_answer_mode: %s", ErrInvalidConfig, c.InboundAnswerMode)
	}
	if c.InboundAnswerMode == InboundAnswerModeAfterMinRingDuration && c.InboundMinRingDuration <= 0 {
		return fmt.Errorf("%w: min_ring_duration is required for answer_after_min_ring_ms", ErrInvalidConfig)
	}
	return nil
}

func (c *Config) GetTransport() Transport {
	if c.Transport == "" {
		return TransportUDP
	}
	return c.Transport
}

func (c *Config) GetSIPURI() string {
	domain := c.Domain
	if domain == "" {
		domain = c.Server
	}
	return fmt.Sprintf("sip:%s@%s:%d", c.Username, domain, c.Port)
}

func (c *Config) GetListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Server, c.Port)
}

// ParseConfigFromVault extracts provider-owned SIP settings from vault.
func ParseConfigFromVault(vaultCredential *protos.VaultCredential) (*Config, error) {
	if vaultCredential == nil || vaultCredential.GetValue() == nil {
		return nil, fmt.Errorf("vault credential is required")
	}

	options := utils.Option(vaultCredential.GetValue().AsMap())
	cfg := &Config{}

	for _, key := range []string{"sip_uri", "host", "host_port"} {
		value, err := options.GetString(key)
		if err != nil || strings.TrimSpace(value) == "" {
			continue
		}

		raw := strings.TrimSpace(value)
		if !strings.HasPrefix(raw, "sip:") && !strings.HasPrefix(raw, "sips:") {
			raw = "sip:" + raw
		}
		var uri sip.Uri
		if err := sip.ParseUri(raw, &uri); err == nil && uri.Host != "" {
			cfg.Server = uri.Host
			if uri.Port > 0 && uri.Port <= maxPort {
				cfg.Port = uri.Port
			}
		}
	}
	if server, err := options.GetString("sip_server"); err == nil && strings.TrimSpace(server) != "" {
		cfg.Server = strings.TrimSpace(server)
	}
	if cfg.Port <= 0 {
		if port, err := options.GetUint32("sip_port"); err == nil && port > 0 && port <= maxPort {
			cfg.Port = int(port)
		}
	}
	if username, err := options.GetString("user"); err == nil && strings.TrimSpace(username) != "" {
		cfg.Username = strings.TrimSpace(username)
	}
	if username, err := options.GetString("sip_username"); err == nil && strings.TrimSpace(username) != "" {
		cfg.Username = strings.TrimSpace(username)
	}
	if password, err := options.GetString("password"); err == nil && strings.TrimSpace(password) != "" {
		cfg.Password = strings.TrimSpace(password)
	}
	if password, err := options.GetString("sip_password"); err == nil && strings.TrimSpace(password) != "" {
		cfg.Password = strings.TrimSpace(password)
	}
	if realm, err := options.GetString("sip_realm"); err == nil && strings.TrimSpace(realm) != "" {
		cfg.Realm = strings.TrimSpace(realm)
	}
	if domain, err := options.GetString("sip_domain"); err == nil && strings.TrimSpace(domain) != "" {
		cfg.Domain = strings.TrimSpace(domain)
	}
	if callerID, err := options.GetString("sip_caller_id"); err == nil && strings.TrimSpace(callerID) != "" {
		cfg.CallerID = strings.TrimSpace(callerID)
	}
	if headers, err := options.GetStringMap("headers"); err == nil && len(headers) > 0 {
		cfg.CustomHeaders = headers
	}
	if headers, err := options.GetStringMap("sip_headers"); err == nil && len(headers) > 0 {
		cfg.CustomHeaders = headers
	}

	return cfg, nil
}
