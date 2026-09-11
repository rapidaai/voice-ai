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
			return &sip_runtime.SIPError{Code: sipStatusServerError, Message: sipMessageMiddlewareChainIncomplete, Err: errMiddlewareChainIncomplete}
		}
		if !validator.NonNil(ctx.Assistant.AssistantPhoneDeployment) {
			return &sip_runtime.SIPError{Code: sipStatusServerError, Message: sipMessageConfigurationResolution, Err: errPhoneDeploymentRequired}
		}
		if !validator.NonNil(m.rapidaClient) || !validator.NonNil(m.rapidaClient.Vault) {
			return &sip_runtime.SIPError{Code: sipStatusServerError, Message: sipMessageVaultResolverUnavailable, Err: errVaultResolverRequired}
		}

		credentialID, err := ctx.Assistant.AssistantPhoneDeployment.GetOptions().GetUint64(credentialIDOptionKey)
		if err != nil {
			return &sip_runtime.SIPError{Code: sipStatusServerError, Message: sipMessageConfigurationResolution, Err: errors.Join(errCredentialIDRequired, err)}
		}

		vaultCredential, err := m.rapidaClient.Vault.GetCredential(m.ctx, ctx.Auth, credentialID)
		if err != nil {
			return &sip_runtime.SIPError{Code: sipStatusServerError, Message: sipMessageConfigurationResolution, Err: errors.Join(errVaultCredentialResolution, err)}
		}

		config, err := sip_runtime.ParseConfigFromVault(vaultCredential)
		if err != nil {
			return &sip_runtime.SIPError{Code: sipStatusServerError, Message: sipMessageConfigurationResolution, Err: errors.Join(errVaultConfigInvalid, err)}
		}

		if did, err := ctx.Assistant.AssistantPhoneDeployment.GetOptions().GetString(phoneOptionKey); err == nil && validator.NotBlank(did) {
			config.CallerID = strings.TrimPrefix(did, phoneNumberPrefix)
		}
		if validator.NonNil(m.applySIPConfigDefaults) {
			m.applySIPConfigDefaults(config)
		}
		ctx.VaultCredential = vaultCredential
		ctx.Config = config
		return nil
	}
}
