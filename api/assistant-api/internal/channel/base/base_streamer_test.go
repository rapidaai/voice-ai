// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_base

import (
	"testing"
	"time"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStreamer(t *testing.T) *BaseStreamer {
	t.Helper()
	logger, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("base-streamer-test"), commons.EnableFile(false))
	require.NoError(t, err)
	streamer := NewBaseStreamerWithChannelCapacity(logger, 2, 2)
	return &streamer
}

func TestNewBaseStreamerInitializesDefaultTransportChannels(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("base-streamer-test"), commons.EnableFile(false))
	require.NoError(t, err)

	streamer := NewBaseStreamer(logger)

	assert.Equal(t, defaultInputChannelCapacity, cap(streamer.InputCh))
	assert.Equal(t, defaultOutputChannelCapacity, cap(streamer.OutputCh))
}

func TestNewBaseStreamerWithChannelCapacityInitializesTransportChannels(t *testing.T) {
	streamer := newTestStreamer(t)

	assert.NotNil(t, streamer.Logger)
	assert.NotNil(t, streamer.Ctx)
	assert.NotNil(t, streamer.Cancel)
	assert.False(t, streamer.Closed)
	assert.Equal(t, criticalChannelCapacity, cap(streamer.CriticalCh))
	assert.Equal(t, 2, cap(streamer.InputCh))
	assert.Equal(t, observabilityChannelCapacity, cap(streamer.LowCh))
	assert.Equal(t, 2, cap(streamer.OutputCh))
}

func TestContextCancelledAfterCancel(t *testing.T) {
	streamer := newTestStreamer(t)
	streamer.Cancel()

	select {
	case <-streamer.Ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("streamer context should be cancelled")
	}
}

func TestInputRoutesCriticalMessages(t *testing.T) {
	streamer := newTestStreamer(t)
	msg := &protos.ConversationDisconnection{}

	streamer.Input(msg)

	select {
	case got := <-streamer.CriticalCh:
		assert.Same(t, msg, got)
	default:
		t.Fatal("expected message on CriticalCh")
	}
}

func TestInputRoutesLowPriorityMessages(t *testing.T) {
	streamer := newTestStreamer(t)
	msg := &protos.ConversationEvent{Name: "health"}

	streamer.Input(msg)

	select {
	case got := <-streamer.LowCh:
		assert.Same(t, msg, got)
	default:
		t.Fatal("expected message on LowCh")
	}
}

func TestInputRoutesNormalMessages(t *testing.T) {
	streamer := newTestStreamer(t)
	msg := &protos.ConversationUserMessage{}

	streamer.Input(msg)

	select {
	case got := <-streamer.InputCh:
		assert.Same(t, msg, got)
	default:
		t.Fatal("expected message on InputCh")
	}
}

func TestInputTracksCriticalDropWhenChannelFull(t *testing.T) {
	streamer := newTestStreamer(t)

	for range cap(streamer.CriticalCh) + 1 {
		streamer.Input(&protos.ConversationDisconnection{})
	}

	stats := streamer.DropStats()
	assert.Equal(t, uint64(1), stats.CriticalInputDropped)
	assert.Zero(t, stats.NormalInputDropped)
	assert.Zero(t, stats.LowInputDropped)
	assert.Zero(t, stats.OutputDropped)
}

func TestInputTracksNormalDropWhenChannelFull(t *testing.T) {
	streamer := newTestStreamer(t)

	for range cap(streamer.InputCh) + 1 {
		streamer.Input(&protos.ConversationUserMessage{})
	}

	stats := streamer.DropStats()
	assert.Equal(t, uint64(1), stats.NormalInputDropped)
	assert.Zero(t, stats.CriticalInputDropped)
	assert.Zero(t, stats.LowInputDropped)
	assert.Zero(t, stats.OutputDropped)
}

func TestInputTracksLowPriorityDropWhenChannelFull(t *testing.T) {
	streamer := newTestStreamer(t)

	for range cap(streamer.LowCh) + 1 {
		streamer.Input(&protos.ConversationEvent{Name: "health"})
	}

	stats := streamer.DropStats()
	assert.Equal(t, uint64(1), stats.LowInputDropped)
	assert.Zero(t, stats.CriticalInputDropped)
	assert.Zero(t, stats.NormalInputDropped)
	assert.Zero(t, stats.OutputDropped)
}

func TestOutputRoutesToOutputChannel(t *testing.T) {
	streamer := newTestStreamer(t)
	msg := &protos.ConversationAssistantMessage{}

	streamer.Output(msg)

	select {
	case got := <-streamer.OutputCh:
		assert.Same(t, msg, got)
	default:
		t.Fatal("expected message on OutputCh")
	}
}

func TestOutputTracksDropWhenChannelFull(t *testing.T) {
	streamer := newTestStreamer(t)

	for range cap(streamer.OutputCh) + 1 {
		streamer.Output(&protos.ConversationAssistantMessage{})
	}

	stats := streamer.DropStats()
	assert.Equal(t, uint64(1), stats.OutputDropped)
	assert.Zero(t, stats.CriticalInputDropped)
	assert.Zero(t, stats.NormalInputDropped)
	assert.Zero(t, stats.LowInputDropped)
}

func TestDisconnectIsIdempotent(t *testing.T) {
	streamer := newTestStreamer(t)

	first := streamer.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER)
	second := streamer.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER)

	require.NotNil(t, first)
	assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_USER, first.GetType())
	assert.NotNil(t, first.GetTime())
	assert.Nil(t, second)
	assert.True(t, streamer.Closed)
}
