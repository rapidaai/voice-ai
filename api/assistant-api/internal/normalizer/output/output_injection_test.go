// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_output_normalizers

import (
	"context"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/require"
)

func TestInjectedMessageInterimEmitsTextWithoutDone(t *testing.T) {
	var packets []internal_type.Packet
	processor := &outputNormalizer{
		onPacket: func(_ context.Context, pkts ...internal_type.Packet) error {
			packets = append(packets, pkts...)
			return nil
		},
	}

	err := processor.Normalize(t.Context(), internal_type.InjectMessagePacket{
		ContextID: "ctx-interim",
		Text:      "continue speaking",
		Interim:   true,
	})

	require.NoError(t, err)
	require.Len(t, packets, 1)
	textPacket, ok := packets[0].(internal_type.TextToSpeechTextPacket)
	require.True(t, ok)
	require.Equal(t, "ctx-interim", textPacket.ContextID)
	require.Equal(t, "continue speaking", textPacket.Text)
}

func TestInjectedMessageStandaloneEmitsTextAndDone(t *testing.T) {
	var packets []internal_type.Packet
	processor := &outputNormalizer{
		onPacket: func(_ context.Context, pkts ...internal_type.Packet) error {
			packets = append(packets, pkts...)
			return nil
		},
	}

	err := processor.Normalize(t.Context(), internal_type.InjectMessagePacket{
		ContextID: "ctx-standalone",
		Text:      "standalone speech",
	})

	require.NoError(t, err)
	require.Len(t, packets, 2)
	textPacket, ok := packets[0].(internal_type.TextToSpeechTextPacket)
	require.True(t, ok)
	require.Equal(t, "ctx-standalone", textPacket.ContextID)
	require.Equal(t, "standalone speech", textPacket.Text)
	donePacket, ok := packets[1].(internal_type.TextToSpeechDonePacket)
	require.True(t, ok)
	require.Equal(t, "ctx-standalone", donePacket.ContextID)
	require.Equal(t, "standalone speech", donePacket.Text)
}
