// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"errors"
	"fmt"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_inbound "github.com/rapidaai/api/assistant-api/sip/internal/inbound"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

// inboundConfig is the resolved tenant config for an inbound SIP call.
// It owns middleware output and the answer policy derived from that config.
type inboundConfig struct {
	config          *Config
	auth            *types.Authentication
	assistant       *internal_assistant_entity.Assistant
	vaultCredential *protos.VaultCredential
	callAddress     CallAddress
	answerPolicy    InboundAnswerPolicy
	setupPhase      InboundSetupPhase
}

// NewInboundConfig runs inbound middleware and returns a call-ready config.
// Failures are already mapped to the SIP response and SLI dimensions.
func NewInboundConfig(server *Server, identity inboundInviteIdentity, mediaOffer inboundMediaOffer) (inboundConfig, *inboundFailure) {
	server.mu.RLock()
	middlewares := append([]Middleware(nil), server.middlewares...)
	server.mu.RUnlock()
	if len(middlewares) == 0 {
		err := fmt.Errorf("no SIP middleware configured")
		return inboundConfig{}, &inboundFailure{
			statusCode:      500,
			class:           inboundFailureConfig,
			responseClass:   internal_inbound.FailureConfig,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_config"},
			lifecycleReason: LifecycleReasonInboundInviteFailed,
			err:             err,
		}
	}

	requestContext := &SIPRequestContext{
		Method:      "INVITE",
		CallID:      identity.callID,
		RequestURI:  identity.requestURI,
		CallAddress: identity.callAddress,
		SDPInfo:     mediaOffer.sdpInfo,
	}
	for _, middleware := range middlewares {
		if err := middleware(requestContext); err != nil {
			var sipErr *SIPError
			if errors.As(err, &sipErr) && sipErr.Code > 0 {
				authErr := fmt.Errorf("%w: %s", ErrAuthRequired, sipErr.Message)
				return inboundConfig{}, &inboundFailure{
					statusCode:      sipErr.Code,
					class:           inboundFailureAuthRequired,
					responseClass:   internal_inbound.FailureAuth,
					reason:          authErr.Error(),
					termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_auth_required"},
					lifecycleReason: LifecycleReasonInboundInviteFailed,
					err:             authErr,
				}
			}

			configErr := fmt.Errorf("SIP middleware failed: %w", err)
			return inboundConfig{}, &inboundFailure{
				statusCode:      500,
				class:           inboundFailureConfig,
				responseClass:   internal_inbound.FailureConfig,
				reason:          configErr.Error(),
				termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_config"},
				lifecycleReason: LifecycleReasonInboundInviteFailed,
				err:             configErr,
			}
		}
	}
	if requestContext.Config == nil {
		err := fmt.Errorf("no SIP config resolved for inbound call")
		return inboundConfig{}, &inboundFailure{
			statusCode:      500,
			class:           inboundFailureConfig,
			responseClass:   internal_inbound.FailureConfig,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_config"},
			lifecycleReason: LifecycleReasonInboundInviteFailed,
			err:             err,
		}
	}

	resolvedConfig := inboundConfig{
		config:          requestContext.Config,
		auth:            requestContext.Auth,
		assistant:       requestContext.Assistant,
		vaultCredential: requestContext.VaultCredential,
		callAddress:     requestContext.CallAddress,
		setupPhase:      InboundSetupPhaseAuthenticated,
	}
	if resolvedConfig.assistant != nil {
		resolvedConfig.setupPhase = InboundSetupPhaseRouted
	}
	if resolvedConfig.config.Server == "" || resolvedConfig.config.Server == "0.0.0.0" {
		resolvedConfig.config.Server = server.listenConfig.GetExternalIP()
	}

	resolvedConfig.answerPolicy = resolvedConfig.config.EffectiveInboundAnswerPolicy(server.effectiveInboundACKTimeout())
	return resolvedConfig, nil
}
