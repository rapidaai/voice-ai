// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"fmt"
	"strings"
)

func (c *Config) ToOutboundConfig() OutboundConfig {
	headers := make(map[string]string, len(c.CustomHeaders))
	for name, value := range c.CustomHeaders {
		headers[name] = value
	}

	return OutboundConfig{
		Mode: OutboundModeTrunkTermination,
		// Route via the outbound proxy when one is configured; otherwise the
		// registrar host doubles as the termination target.
		Address:         c.GetOutboundTarget(),
		Port:            c.Port,
		Transport:       c.GetTransport(),
		Domain:          c.Domain,
		RingingTimeout:  c.InviteTimeout,
		MaxCallDuration: c.SessionTimeout,
		Auth: SIPAuthConfig{
			Username: c.GetAuthUsername(),
			Password: c.Password,
			Realm:    c.Realm,
		},
		Headers: headers,
	}
}

func NewOutboundInviteRequest(cfg *Config, toUser string, fromUser string) (OutboundInviteRequest, error) {
	if cfg == nil {
		return OutboundInviteRequest{}, fmt.Errorf("%w: config is required", ErrInvalidConfig)
	}

	request := OutboundInviteRequest{
		Config: cfg.ToOutboundConfig(),
		Identity: OutboundCallIdentity{
			ToUser:   strings.TrimSpace(toUser),
			FromUser: strings.TrimSpace(fromUser),
		},
	}
	if err := request.Validate(); err != nil {
		return OutboundInviteRequest{}, err
	}
	return request, nil
}

func (r OutboundInviteRequest) Validate() error {
	switch r.Config.Mode {
	case OutboundModeTrunkTermination:
	default:
		return fmt.Errorf("%w: unsupported outbound mode %q", ErrInvalidConfig, r.Config.Mode)
	}

	if r.Config.Address == "" {
		return fmt.Errorf("%w: outbound address is required", ErrInvalidConfig)
	}
	if strings.HasPrefix(r.Config.Address, "sip:") || strings.HasPrefix(r.Config.Address, "sips:") {
		return fmt.Errorf("%w: outbound address must be a host without SIP scheme", ErrInvalidConfig)
	}
	if strings.ContainsAny(r.Config.Address, ";=") {
		return fmt.Errorf("%w: outbound address must not contain URI parameters", ErrInvalidConfig)
	}
	if r.Config.Port <= 0 || r.Config.Port > 65535 {
		return fmt.Errorf("%w: outbound port must be between 1 and 65535", ErrInvalidConfig)
	}
	if !r.Config.Transport.IsValid() {
		return fmt.Errorf("%w: invalid outbound transport: %s", ErrInvalidConfig, r.Config.Transport)
	}
	if r.Identity.ToUser == "" {
		return fmt.Errorf("%w: outbound destination user is required", ErrInvalidConfig)
	}
	if strings.Contains(r.Identity.ToUser, "@") {
		return fmt.Errorf("%w: outbound destination must be a phone number or SIP user, not a full SIP URI", ErrInvalidConfig)
	}
	if r.Identity.FromUser == "" {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, ErrOutboundFromUserRequired)
	}
	return nil
}

func outboundAuthMissingForChallenge(auth SIPAuthConfig, statusCode int) bool {
	if statusCode != 401 && statusCode != 407 {
		return false
	}
	return auth.Username == "" || auth.Password == ""
}
