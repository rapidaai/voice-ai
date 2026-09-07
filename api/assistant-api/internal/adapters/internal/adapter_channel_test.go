// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"testing"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	policychannel "github.com/rapidaai/pkg/channel"
)

func receiveEnvelope(t testing.TB, ch *policychannel.Channel[adapter_channel.Envelope]) adapter_channel.Envelope {
	t.Helper()
	envelope, err := ch.TryReceive()
	if err != nil {
		t.Fatalf("receive envelope: %v", err)
	}
	return envelope
}
