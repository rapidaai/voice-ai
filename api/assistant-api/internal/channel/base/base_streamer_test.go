// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_base

import (
	"testing"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInput_RecognitionOverflowEvictsOldestAudio(tester *testing.T) {
	streamer := New(WithInputChannelCapacity(1), WithOutputChannelCapacity(1))
	defer streamer.Cancel()
	first := &protos.ConversationUserMessage{Message: &protos.ConversationUserMessage_Audio{Audio: []byte{1, 2}}}
	latest := &protos.ConversationUserMessage{Message: &protos.ConversationUserMessage_Audio{Audio: []byte{3, 4}}}
	streamer.Input(first)
	streamer.Input(latest)
	message, err := streamer.Recv()
	require.NoError(tester, err)
	require.Same(tester, latest, message)
}

func TestInput_AudioOverflowDoesNotEvictInitialization(tester *testing.T) {
	streamer := New(WithInputChannelCapacity(1), WithOutputChannelCapacity(1))
	defer streamer.Cancel()
	initialization := &protos.ConversationInitialization{}
	latestAudio := &protos.ConversationUserMessage{Message: &protos.ConversationUserMessage_Audio{Audio: []byte{3, 4}}}

	streamer.Input(initialization)
	streamer.Input(&protos.ConversationUserMessage{Message: &protos.ConversationUserMessage_Audio{Audio: []byte{1, 2}}})
	streamer.Input(latestAudio)

	message, err := streamer.Recv()
	require.NoError(tester, err)
	require.Same(tester, initialization, message)
	message, err = streamer.Recv()
	require.NoError(tester, err)
	require.Same(tester, latestAudio, message)
}

func TestInputRoutesBridgeAudioToLowPriority(tester *testing.T) {
	streamer := New(WithInputChannelCapacity(2), WithOutputChannelCapacity(1))
	defer streamer.Cancel()
	streamer.Input(&protos.ConversationBridgeUserAudio{Audio: []byte{1, 2}})
	streamer.Input(&protos.ConversationBridgeOperatorAudio{Audio: []byte{3, 4}})
	require.Empty(tester, streamer.InputCh)
	require.Len(tester, streamer.LowCh, 2)
}

func newTestStreamer(t *testing.T) *BaseStreamer {
	t.Helper()
	logger, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("base-streamer-test"), commons.EnableFile(false))
	require.NoError(t, err)
	streamer := New(
		WithLogger(logger),
		WithInputChannelCapacity(2),
		WithOutputChannelCapacity(2),
	)
	return &streamer
}

func TestNewBaseStreamerInitializesDefaultTransportChannels(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("base-streamer-test"), commons.EnableFile(false))
	require.NoError(t, err)

	streamer := New(WithLogger(logger))

	assert.Equal(t, defaultInputChannelCapacity, cap(streamer.InputCh))
	assert.Equal(t, defaultOutputChannelCapacity, cap(streamer.OutputCh))
}

func TestNewWithChannelCapacityOptionsInitializesTransportChannels(t *testing.T) {
	streamer := newTestStreamer(t)

	assert.NotNil(t, streamer.Logger)
	assert.NotNil(t, streamer.Ctx)
	assert.NotNil(t, streamer.Cancel)
	assert.False(t, streamer.Closed)
	assert.Equal(t, criticalChannelCapacity, cap(streamer.CriticalCh))
	assert.Equal(t, 2, cap(streamer.InputCh))
	assert.Equal(t, lowPriorityChannelCapacity, cap(streamer.LowCh))
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
	messages := []internal_type.Stream{
		&protos.ConversationDisconnection{},
		&protos.ConversationInitialization{},
		&protos.ConversationConfiguration{},
		&protos.ConversationUserMessage{Message: &protos.ConversationUserMessage_Text{Text: "hello"}},
	}

	for _, msg := range messages {
		streamer.Input(msg)
		select {
		case got := <-streamer.CriticalCh:
			assert.Same(t, msg, got)
		default:
			t.Fatal("expected message on CriticalCh")
		}
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
	msg := &protos.ConversationUserMessage{
		Message: &protos.ConversationUserMessage_Audio{Audio: []byte{1, 2}},
	}

	streamer.Input(msg)

	select {
	case got := <-streamer.InputCh:
		assert.Same(t, msg, got)
	default:
		t.Fatal("expected message on InputCh")
	}
}

func TestRecvPrefersRealtimeInputOverLowPriority(t *testing.T) {
	streamer := newTestStreamer(t)
	lowPriority := &protos.ConversationBridgeUserAudio{Audio: []byte{1}}
	realtime := &protos.ConversationUserMessage{
		Message: &protos.ConversationUserMessage_Audio{Audio: []byte{2}},
	}

	streamer.Input(lowPriority)
	streamer.Input(realtime)

	message, err := streamer.Recv()
	require.NoError(t, err)
	assert.Same(t, realtime, message)

	message, err = streamer.Recv()
	require.NoError(t, err)
	assert.Same(t, lowPriority, message)
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
