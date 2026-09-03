// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"

type BridgeEndReason = internal_core.BridgeEndReason

const (
	BridgeEndInboundBye  = internal_core.BridgeEndInboundBye
	BridgeEndOutboundBye = internal_core.BridgeEndOutboundBye
	BridgeEndContext     = internal_core.BridgeEndContext
	BridgeEndTimeout     = internal_core.BridgeEndTimeout
)
