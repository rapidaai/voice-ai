// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_base

import (
	"io"
	"testing"
	"testing/synctest"
	"time"

	"github.com/rapidaai/pkg/channel"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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
	userAudio := &protos.ConversationBridgeUserAudio{Audio: []byte{1, 2}}
	operatorAudio := &protos.ConversationBridgeOperatorAudio{Audio: []byte{3, 4}}
	streamer.Input(userAudio)
	streamer.Input(operatorAudio)
	require.Zero(tester, streamer.InputCh.Len())
	require.Equal(tester, 2, streamer.LowCh.Len())
	message, err := streamer.LowCh.TryReceive()
	require.NoError(tester, err)
	require.Same(tester, userAudio, message)
	message, err = streamer.LowCh.TryReceive()
	require.NoError(tester, err)
	require.Same(tester, operatorAudio, message)
}

func TestInputPlaybackCompleteWaitsForCriticalCapacity(t *testing.T) {
	streamer := New(WithInputChannelCapacity(1))
	defer streamer.Cancel()
	for range streamer.CriticalCh.Capacity() {
		streamer.Input(&protos.ConversationInitialization{})
	}
	completion := &protos.ConversationPlaybackComplete{Id: "response-1"}
	started := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		close(started)
		streamer.Input(completion)
		close(finished)
	}()
	<-started
	select {
	case <-finished:
		t.Fatal("completion was dropped while the critical queue was full")
	case <-time.After(10 * time.Millisecond):
	}
	_, err := streamer.Recv()
	require.NoError(t, err)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("completion did not use the available critical queue slot")
	}
	for range streamer.CriticalCh.Capacity() - 1 {
		_, err = streamer.Recv()
		require.NoError(t, err)
	}
	message, err := streamer.Recv()
	require.NoError(t, err)
	require.Same(t, completion, message)
	require.Zero(t, streamer.InputCh.Len())
}

func TestInputPlaybackCompleteUnblocksOnCancellation(t *testing.T) {
	streamer := New()
	defer streamer.Cancel()
	for range streamer.CriticalCh.Capacity() {
		streamer.Input(&protos.ConversationInitialization{})
	}
	finished := make(chan struct{})
	go func() {
		streamer.Input(&protos.ConversationPlaybackComplete{Id: "response-1"})
		close(finished)
	}()
	streamer.Cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("completion delivery remained blocked after cancellation")
	}
	require.Equal(t, streamer.CriticalCh.Capacity(), streamer.CriticalCh.Len())
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

	assert.Equal(t, defaultInputChannelCapacity, streamer.InputCh.Capacity())
	assert.Equal(t, defaultOutputChannelCapacity, streamer.OutputCh.Capacity())
}

func TestNewWithChannelCapacityOptionsInitializesTransportChannels(t *testing.T) {
	streamer := newTestStreamer(t)

	assert.NotNil(t, streamer.Logger)
	assert.NotNil(t, streamer.Ctx)
	assert.NotNil(t, streamer.Cancel)
	assert.False(t, streamer.Closed)
	assert.Equal(t, criticalChannelCapacity, streamer.CriticalCh.Capacity())
	assert.Equal(t, 2, streamer.InputCh.Capacity())
	assert.Equal(t, lowPriorityChannelCapacity, streamer.LowCh.Capacity())
	assert.Equal(t, 2, streamer.OutputCh.Capacity())
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
	messages := []proto.Message{
		&protos.ConversationDisconnection{},
		&protos.ConversationInitialization{},
		&protos.ConversationConfiguration{},
		&protos.ConversationUserMessage{Message: &protos.ConversationUserMessage_Text{Text: "hello"}},
	}

	for _, msg := range messages {
		streamer.Input(msg)
		got, err := streamer.CriticalCh.TryReceive()
		require.NoError(t, err)
		assert.Same(t, msg, got)
	}
}

func TestInputRoutesLowPriorityMessages(t *testing.T) {
	streamer := newTestStreamer(t)
	msg := &protos.ConversationEvent{Name: "health"}

	streamer.Input(msg)

	got, err := streamer.LowCh.TryReceive()
	require.NoError(t, err)
	assert.Same(t, msg, got)
}

func TestInputRoutesNormalMessages(t *testing.T) {
	streamer := newTestStreamer(t)
	msg := &protos.ConversationUserMessage{
		Message: &protos.ConversationUserMessage_Audio{Audio: []byte{1, 2}},
	}

	streamer.Input(msg)

	got, err := streamer.InputCh.TryReceive()
	require.NoError(t, err)
	assert.Same(t, msg, got)
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

func TestRecvReturnsEOFAfterCancel(t *testing.T) {
	streamer := newTestStreamer(t)
	streamer.Cancel()

	message, err := streamer.Recv()

	require.ErrorIs(t, err, io.EOF)
	require.Nil(t, message)
}

func TestOutputRoutesToOutputChannel(t *testing.T) {
	streamer := newTestStreamer(t)
	msg := &protos.ConversationAssistantMessage{}

	streamer.Output(msg)

	got, err := streamer.OutputCh.TryReceive()
	require.NoError(t, err)
	assert.Same(t, msg, got)
}

func TestReliableQueuesWaitForCapacity(t *testing.T) {
	for _, queueName := range []string{"critical", "output"} {
		t.Run(queueName, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				streamer := New(WithOutputChannelCapacity(1))
				defer streamer.Cancel()
				queue := streamer.CriticalCh
				send := streamer.Input
				if queueName == "output" {
					queue = streamer.OutputCh
					send = streamer.Output
				}
				first := &protos.ConversationInitialization{}
				for range queue.Capacity() {
					send(first)
				}
				last := &protos.ConversationToolCallResult{Id: "pending"}
				finished := make(chan struct{})
				go func() {
					send(last)
					close(finished)
				}()
				synctest.Wait()
				select {
				case <-finished:
					t.Fatal("reliable send must wait for capacity")
				default:
				}
				message, err := queue.TryReceive()
				require.NoError(t, err)
				require.Same(t, first, message)
				synctest.Wait()
				select {
				case <-finished:
				default:
					t.Fatal("reliable send did not resume after capacity became available")
				}
				for range queue.Capacity() - 1 {
					message, err = queue.TryReceive()
					require.NoError(t, err)
					require.Same(t, first, message)
				}
				message, err = queue.TryReceive()
				require.NoError(t, err)
				require.Same(t, last, message)
			})
		})
	}
}

func TestReliableQueuesUnblockOnShutdown(t *testing.T) {
	for _, queueName := range []string{"critical", "output"} {
		for _, shutdown := range []string{"cancel", "close"} {
			t.Run(queueName+"/"+shutdown, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					streamer := New(WithOutputChannelCapacity(1))
					defer streamer.Cancel()
					queue := streamer.CriticalCh
					send := streamer.Input
					if queueName == "output" {
						queue = streamer.OutputCh
						send = streamer.Output
					}
					for range queue.Capacity() {
						send(&protos.ConversationInitialization{})
					}
					finished := make(chan struct{})
					go func() {
						send(&protos.ConversationToolCallResult{})
						close(finished)
					}()
					synctest.Wait()
					if shutdown == "cancel" {
						streamer.Cancel()
					} else {
						queue.Close()
					}
					synctest.Wait()
					select {
					case <-finished:
					default:
						t.Fatal("reliable send remained blocked after shutdown")
					}
					require.Equal(t, queue.Capacity(), queue.Len())
				})
			})
		}
	}
}

func TestLowPriorityRejectsNewestWhenFull(t *testing.T) {
	streamer := New()
	defer streamer.Cancel()
	first := &protos.ConversationMetric{}
	for range streamer.LowCh.Capacity() {
		streamer.Input(first)
	}
	streamer.Input(&protos.ConversationEvent{Name: "rejected"})
	for range streamer.LowCh.Capacity() {
		message, err := streamer.LowCh.TryReceive()
		require.NoError(t, err)
		require.Same(t, first, message)
	}
	_, err := streamer.LowCh.TryReceive()
	require.ErrorIs(t, err, channel.ErrEmpty)
}

func TestDisconnectionDeliveryHasBoundedWait(t *testing.T) {
	for _, queueName := range []string{"critical", "output"} {
		t.Run(queueName, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				streamer := New(WithOutputChannelCapacity(1))
				defer streamer.Cancel()
				queue := streamer.CriticalCh
				send := streamer.Input
				if queueName == "output" {
					queue = streamer.OutputCh
					send = streamer.Output
				}
				for range queue.Capacity() {
					send(&protos.ConversationInitialization{})
				}
				started := time.Now()
				send(&protos.ConversationDisconnection{})
				require.Equal(t, disconnectionDeliveryTimeout, time.Since(started))
				require.NoError(t, streamer.Ctx.Err())
				require.Equal(t, queue.Capacity(), queue.Len())
			})
		})
	}
}

func TestRecvWakesForEachQueue(t *testing.T) {
	for _, queueName := range []string{"critical", "input", "low"} {
		t.Run(queueName, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				streamer := New()
				defer streamer.Cancel()
				var message proto.Message = &protos.ConversationInitialization{}
				if queueName == "input" {
					message = &protos.ConversationUserMessage{Message: &protos.ConversationUserMessage_Audio{Audio: []byte{1}}}
				} else if queueName == "low" {
					message = &protos.ConversationEvent{}
				}
				var received proto.Message
				var receiveError error
				finished := make(chan struct{})
				go func() {
					received, receiveError = streamer.Recv()
					close(finished)
				}()
				synctest.Wait()
				streamer.Input(message)
				synctest.Wait()
				select {
				case <-finished:
				default:
					t.Fatal("Recv did not wake for queued message")
				}
				require.NoError(t, receiveError)
				require.Same(t, message, received)
			})
		})
	}
}

func TestRecvReturnsEOFForClosedQueues(t *testing.T) {
	for _, queueName := range []string{"critical", "input", "low"} {
		t.Run(queueName, func(t *testing.T) {
			streamer := New()
			defer streamer.Cancel()
			switch queueName {
			case "critical":
				streamer.CriticalCh.Close()
			case "input":
				streamer.InputCh.Close()
			case "low":
				streamer.LowCh.Close()
			}
			message, err := streamer.Recv()
			require.ErrorIs(t, err, io.EOF)
			require.Nil(t, message)
		})
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
