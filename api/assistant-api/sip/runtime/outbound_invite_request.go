// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"fmt"
	"strings"

	sip_config "github.com/rapidaai/api/assistant-api/sip/config"
)

func toOutboundConfig(config *sip_config.Config) *OutboundConfig {
	if config == nil {
		return nil
	}

	headers := make(map[string]string, len(config.CustomHeaders))
	for name, value := range config.CustomHeaders {
		headers[name] = value
	}

	return &OutboundConfig{
		Mode:            OutboundModeTrunkTermination,
		Address:         config.Server,
		Port:            config.Port,
		Transport:       config.GetTransport(),
		Domain:          config.Domain,
		RingingTimeout:  config.InviteTimeout,
		MaxCallDuration: config.SessionTimeout,
		Auth: SIPAuthConfig{
			Username: config.Username,
			Password: config.Password,
			Realm:    config.Realm,
		},
		Headers:             headers,
		MediaTimeoutInitial: config.MediaTimeoutInitial,
		MediaTimeout:        config.MediaTimeout,
	}
}

func NewOutboundInviteRequest(config *sip_config.Config, toUser string, fromUser string) (OutboundInviteRequest, error) {
	if config == nil {
		return OutboundInviteRequest{}, fmt.Errorf("%w: config is required", sip_config.ErrInvalidConfig)
	}

	request := OutboundInviteRequest{
		Config: toOutboundConfig(config),
		Address: CallAddress{
			To:   strings.TrimSpace(toUser),
			From: strings.TrimSpace(fromUser),
		},
	}
	if err := request.Validate(); err != nil {
		return OutboundInviteRequest{}, err
	}
	return request, nil
}

func (request OutboundInviteRequest) Validate() error {
	if request.Config == nil {
		return fmt.Errorf("%w: outbound config is required", sip_config.ErrInvalidConfig)
	}

	switch request.Config.Mode {
	case OutboundModeTrunkTermination:
	default:
		return fmt.Errorf("%w: unsupported outbound mode %q", sip_config.ErrInvalidConfig, request.Config.Mode)
	}

	if request.Config.Address == "" {
		return fmt.Errorf("%w: outbound address is required", sip_config.ErrInvalidConfig)
	}
	if strings.HasPrefix(request.Config.Address, "sip:") || strings.HasPrefix(request.Config.Address, "sips:") {
		return fmt.Errorf("%w: outbound address must be a host without SIP scheme", sip_config.ErrInvalidConfig)
	}
	if strings.ContainsAny(request.Config.Address, ";=") {
		return fmt.Errorf("%w: outbound address must not contain URI parameters", sip_config.ErrInvalidConfig)
	}
	if request.Config.Port <= 0 || request.Config.Port > 65535 {
		return fmt.Errorf("%w: outbound port must be between 1 and 65535", sip_config.ErrInvalidConfig)
	}
	if !request.Config.Transport.IsValid() {
		return fmt.Errorf("%w: invalid outbound transport: %s", sip_config.ErrInvalidConfig, request.Config.Transport)
	}
	if request.Address.To == "" {
		return fmt.Errorf("%w: outbound destination user is required", sip_config.ErrInvalidConfig)
	}
	if strings.Contains(request.Address.To, "@") {
		return fmt.Errorf("%w: outbound destination must be a phone number or SIP user, not a full SIP URI", sip_config.ErrInvalidConfig)
	}
	if request.Address.From == "" {
		return fmt.Errorf("%w: %w", sip_config.ErrInvalidConfig, ErrOutboundFromUserRequired)
	}
	return nil
}

func outboundAuthMissingForChallenge(auth SIPAuthConfig, statusCode int) bool {
	if statusCode != 401 && statusCode != 407 {
		return false
	}
	return auth.Username == "" || auth.Password == ""
}
