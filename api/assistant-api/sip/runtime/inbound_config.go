// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"errors"
	"fmt"
	"maps"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
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

// resolveConfig runs inbound middleware and stores the call-ready config.
// Failures are already mapped to the SIP response and SLI dimensions.
func (inboundCall *Inbound) resolveConfig() *inboundFailure {
	server := inboundCall.server
	server.mu.RLock()
	middlewares := append([]Middleware(nil), server.middlewares...)
	server.mu.RUnlock()
	if len(middlewares) == 0 {
		err := fmt.Errorf("no SIP middleware configured")
		return &inboundFailure{
			statusCode:      500,
			class:           inboundFailureConfig,
			responseClass:   inboundFailureConfig,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_config"},
			lifecycleReason: LifecycleReasonInboundInviteFailed,
			err:             err,
		}
	}

	middlewareCallAddress := inboundCall.identity.callAddress
	middlewareCallAddress.Headers = maps.Clone(middlewareCallAddress.Headers)
	requestContext := &SIPRequestContext{
		Method:      "INVITE",
		CallID:      inboundCall.identity.callID,
		RequestURI:  inboundCall.identity.requestURI,
		CallAddress: middlewareCallAddress,
		SDPInfo:     inboundCall.mediaOffer.sdpInfo,
	}
	for _, middleware := range middlewares {
		if err := middleware(requestContext); err != nil {
			if errors.Is(err, ErrInvalidConfig) {
				configErr := fmt.Errorf("SIP middleware failed: %w", err)
				return &inboundFailure{
					statusCode:      500,
					class:           inboundFailureConfig,
					responseClass:   inboundFailureConfig,
					reason:          configErr.Error(),
					termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_config"},
					lifecycleReason: LifecycleReasonInboundInviteFailed,
					err:             configErr,
				}
			}

			var sipErr *SIPError
			if errors.As(err, &sipErr) && sipErr.Code > 0 {
				authErr := fmt.Errorf("%w: %s", ErrAuthRequired, sipErr.Message)
				return &inboundFailure{
					statusCode:      sipErr.Code,
					class:           inboundFailureAuthRequired,
					responseClass:   inboundFailureAuth,
					reason:          authErr.Error(),
					termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_auth_required"},
					lifecycleReason: LifecycleReasonInboundInviteFailed,
					err:             authErr,
				}
			}

			configErr := fmt.Errorf("SIP middleware failed: %w", err)
			return &inboundFailure{
				statusCode:      500,
				class:           inboundFailureConfig,
				responseClass:   inboundFailureConfig,
				reason:          configErr.Error(),
				termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_config"},
				lifecycleReason: LifecycleReasonInboundInviteFailed,
				err:             configErr,
			}
		}
	}
	if requestContext.Config == nil {
		err := fmt.Errorf("no SIP config resolved for inbound call")
		return &inboundFailure{
			statusCode:      500,
			class:           inboundFailureConfig,
			responseClass:   inboundFailureConfig,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationServerError, Reason: "inbound_config"},
			lifecycleReason: LifecycleReasonInboundInviteFailed,
			err:             err,
		}
	}

	callAddress := inboundCall.identity.callAddress
	callAddress.To = requestContext.CallAddress.To
	inboundCall.resolvedConfig = inboundConfig{
		config:          requestContext.Config,
		auth:            requestContext.Auth,
		assistant:       requestContext.Assistant,
		vaultCredential: requestContext.VaultCredential,
		callAddress:     callAddress,
		setupPhase:      InboundSetupPhaseAuthenticated,
	}
	if inboundCall.resolvedConfig.assistant != nil {
		inboundCall.resolvedConfig.setupPhase = InboundSetupPhaseRouted
	}
	if inboundCall.resolvedConfig.config.Server == "" || inboundCall.resolvedConfig.config.Server == "0.0.0.0" {
		inboundCall.resolvedConfig.config.Server = server.listenConfig.GetExternalIP()
	}

	inboundCall.resolvedConfig.answerPolicy = inboundCall.resolvedConfig.config.EffectiveInboundAnswerPolicy(server.effectiveInboundACKTimeout())
	inboundCall.identity.callAddress = callAddress
	return nil
}
