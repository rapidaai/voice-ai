// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

import (
	"context"
	"errors"
	"strings"

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/validator"
)

func NewVaultMiddleware(options ...func(*middlewareOption)) sip_runtime.Middleware {
	m := &middlewareOption{ctx: context.Background()}
	for _, option := range options {
		if validator.NonNil(option) {
			option(m)
		}
	}
	return func(ctx *sip_runtime.SIPRequestContext) error {
		if !validator.NonNil(ctx.Auth) || !validator.NonNil(ctx.Assistant) {
			return &sip_runtime.SIPError{Code: 500, Message: "Middleware chain incomplete", Err: errors.Join(sip_runtime.ErrInvalidConfig, sip_runtime.ErrMiddlewareChainIncomplete)}
		}
		if !validator.NonNil(ctx.Assistant.AssistantPhoneDeployment) {
			return &sip_runtime.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: errors.Join(sip_runtime.ErrInvalidConfig, sip_runtime.ErrPhoneDeploymentRequired)}
		}
		if !validator.NonNil(m.rapidaClient) || !validator.NonNil(m.rapidaClient.Vault) {
			return &sip_runtime.SIPError{Code: 500, Message: "SIP vault resolver not configured", Err: errors.Join(sip_runtime.ErrInvalidConfig, sip_runtime.ErrVaultResolverRequired)}
		}

		credentialID, err := ctx.Assistant.AssistantPhoneDeployment.GetOptions().GetUint64("rapida.credential_id")
		if err != nil {
			return &sip_runtime.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: errors.Join(sip_runtime.ErrInvalidConfig, sip_runtime.ErrCredentialIDRequired, err)}
		}

		vaultCredential, err := m.rapidaClient.Vault.GetCredential(m.ctx, ctx.Auth, credentialID)
		if err != nil {
			return &sip_runtime.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: errors.Join(sip_runtime.ErrInvalidConfig, sip_runtime.ErrVaultCredentialResolution, err)}
		}

		config, err := sip_runtime.ParseConfigFromVault(vaultCredential)
		if err != nil {
			return &sip_runtime.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: errors.Join(sip_runtime.ErrInvalidConfig, sip_runtime.ErrVaultConfigInvalid, err)}
		}

		if did, err := ctx.Assistant.AssistantPhoneDeployment.GetOptions().GetString("phone"); err == nil && validator.NotBlank(did) {
			config.CallerID = strings.TrimPrefix(did, "+")
		}
		if validator.NonNil(m.applySIPConfigDefaults) {
			m.applySIPConfigDefaults(config)
		}
		ctx.VaultCredential = vaultCredential
		ctx.Config = config
		return nil
	}
}
