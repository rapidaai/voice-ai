// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import "context"

func (s *Server) BridgeTransfer(ctx context.Context, inbound, outbound *Session, onOperatorAudio func([]byte)) (BridgeEndReason, error) {
	return s.inner.BridgeTransfer(ctx, inbound.unwrap(), outbound.unwrap(), onOperatorAudio)
}
