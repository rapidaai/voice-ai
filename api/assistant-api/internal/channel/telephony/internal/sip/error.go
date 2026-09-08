// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import "errors"

var (
	ErrInboundCallerMissing           = errors.New("missing caller information")
	ErrSIPServerNotInitialized        = errors.New("sip server not initialized")
	ErrSIPServerNotRunning            = errors.New("sip server not running")
	ErrOutboundHealthGateFailed       = errors.New("sip outbound health gate failed")
	ErrSessionRequired                = errors.New("sip session is required; standalone server mode is not supported")
	ErrLifecycleControllerRequired    = errors.New("sip lifecycle controller is required")
	ErrProviderAudioConversionFailed  = errors.New("audio conversion to 16kHz linear16 failed")
	ErrAssistantAudioConversionFailed = errors.New("audio conversion to mulaw 8kHz failed")
)
