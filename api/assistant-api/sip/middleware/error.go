// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

import (
	"errors"

	sip_config "github.com/rapidaai/api/assistant-api/sip/config"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
)

var (
	errMiddlewareChainIncomplete = errors.Join(sip_config.ErrInvalidConfig, sip_runtime.ErrMiddlewareChainIncomplete)
	errPhoneDeploymentRequired   = errors.Join(sip_config.ErrInvalidConfig, sip_runtime.ErrPhoneDeploymentRequired)
	errVaultResolverRequired     = errors.Join(sip_config.ErrInvalidConfig, sip_runtime.ErrVaultResolverRequired)
	errCredentialIDRequired      = errors.Join(sip_config.ErrInvalidConfig, sip_runtime.ErrCredentialIDRequired)
	errVaultCredentialResolution = errors.Join(sip_config.ErrInvalidConfig, sip_runtime.ErrVaultCredentialResolution)
	errVaultConfigInvalid        = errors.Join(sip_config.ErrInvalidConfig, sip_runtime.ErrVaultConfigInvalid)
)
