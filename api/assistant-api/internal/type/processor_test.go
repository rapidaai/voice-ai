// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_type

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

type testTextProcessor struct{}

func (testTextProcessor) Process(text string) string { return "prepared:" + text }

type testPacketProcessor struct {
	err error
}

func (testPacketProcessor) Initialize(context.Context, Communication, *protos.ConversationInitialization) error {
	return nil
}

func (p testPacketProcessor) Process(context.Context, ...Packet) error { return p.err }
func (testPacketProcessor) Close(context.Context) error                { return nil }

func TestTextProcessorContract(t *testing.T) {
	var processor TextProcessor = testTextProcessor{}
	require.Equal(t, "prepared:hello", processor.Process("hello"))
}

func TestPacketProcessorContract(t *testing.T) {
	wantErr := errors.New("process failed")
	var processor PacketProcessor = testPacketProcessor{err: wantErr}

	require.NoError(t, processor.Initialize(t.Context(), nil, nil))
	require.ErrorIs(t, processor.Process(t.Context()), wantErr)
	require.NoError(t, processor.Close(t.Context()))
}
