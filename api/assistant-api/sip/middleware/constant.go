// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

const (
	sipStatusNotFound    = 404
	sipStatusServerError = 500
)

const (
	sipMessageInvalidRoute                 = "Invalid SIP route"
	sipMessageUnsupportedRoute             = "Unsupported SIP route"
	sipMessageAssistantResolverUnavailable = "SIP assistant resolver not configured"
	sipMessageAssistantRouteNotFound       = "No assistant found for this SIP route"
	sipMessageConfigurationResolution      = "Failed to resolve SIP configuration"
	sipMessageMiddlewareChainIncomplete    = "Middleware chain incomplete"
	sipMessageVaultResolverUnavailable     = "SIP vault resolver not configured"
)

const (
	phoneOptionKey = "phone"
	// #nosec G101, this is a metadata key name, not a credential value.
	credentialIDOptionKey = "rapida.credential_id"
	phoneNumberPrefix     = "+"
)
