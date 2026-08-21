// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

import (
	"context"
	"fmt"
	"strings"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
)

func NewVaultMiddleware(options ...func(*middlewareOption)) sip_infra.Middleware {
	m := &middlewareOption{ctx: context.Background()}
	for _, option := range options {
		if validator.NonNil(option) {
			option(m)
		}
	}
	return func(ctx *sip_infra.SIPRequestContext) error {
		auth := ctx.Auth
		assistant := ctx.Assistant

		if !validator.NonNil(auth) || !validator.NonNil(assistant) {
			return &sip_infra.SIPError{Code: 500, Message: "Middleware chain incomplete", Err: sip_infra.ErrInvalidConfig}
		}
		if !validator.NonNil(assistant.AssistantPhoneDeployment) {
			m.logger.Error("SIP: failed to resolve config",
				"call_id", ctx.CallID,
				"method", ctx.Method,
				"error", "assistant has no phone deployment configured")
			return &sip_infra.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: sip_infra.ErrInvalidConfig}
		}
		if !validator.NonNil(m.vaultClient) {
			return &sip_infra.SIPError{Code: 500, Message: "SIP vault resolver not configured", Err: sip_infra.ErrInvalidConfig}
		}

		opts := assistant.AssistantPhoneDeployment.GetOptions()
		credentialID, err := opts.GetUint64("rapida.credential_id")
		if err != nil {
			m.logger.Error("SIP: failed to resolve config",
				"call_id", ctx.CallID,
				"method", ctx.Method,
				"error", fmt.Errorf("no credential_id in phone deployment: %w", err))
			return &sip_infra.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: sip_infra.ErrInvalidConfig}
		}

		vaultCred, err := m.vaultClient.GetCredential(m.ctx, auth, credentialID)
		if err != nil {
			m.logger.Error("SIP: failed to resolve config",
				"call_id", ctx.CallID,
				"method", ctx.Method,
				"error", fmt.Errorf("failed to fetch vault credential %d: %w", credentialID, err))
			return &sip_infra.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: sip_infra.ErrInvalidConfig}
		}

		sipConfig, err := sip_infra.ParseConfigFromVault(vaultCred)
		if err != nil {
			m.logger.Error("SIP: failed to resolve config",
				"call_id", ctx.CallID,
				"method", ctx.Method,
				"error", fmt.Errorf("failed to parse SIP config from vault: %w", err))
			return &sip_infra.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: sip_infra.ErrInvalidConfig}
		}

		if did, err := opts.GetString("phone"); err == nil && validator.NotBlank(did) {
			sipConfig.CallerID = strings.TrimPrefix(did, "+")
		}
		if validator.NonNil(m.applySIPConfigDefaults) {
			m.applySIPConfigDefaults(sipConfig)
		}

		var orgID uint64
		if delegatedContext, err := types.ResolveDelegatedContext(auth); err == nil {
			orgID = delegatedContext.OrganizationID
		}
		m.logger.Infow("SIP request authenticated",
			"call_id", ctx.CallID,
			"method", ctx.Method,
			"assistant_id", assistant.Id,
			"org_id", orgID)

		ctx.Auth = auth
		ctx.Assistant = assistant
		ctx.VaultCredential = vaultCred
		ctx.Config = sipConfig
		return nil
	}
}
