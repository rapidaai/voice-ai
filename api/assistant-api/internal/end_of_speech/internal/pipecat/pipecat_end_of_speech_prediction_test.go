// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEOS_StoppedTurnCompletesWhilePredictionIsBlocked(t *testing.T) {
	for _, finalBeforePrediction := range []bool{true, false} {
		name := "final before prediction"
		if !finalBeforePrediction {
			name = "final during prediction"
		}
		t.Run(name, func(t *testing.T) {
			predictionStarted := make(chan struct{})
			finishPrediction := make(chan struct{})
			predictionReturned := make(chan struct{})
			completed := make(chan internal_type.EndOfSpeechPacket, 4)
			endOfSpeech := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- packet
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": 100.0,
				"microphone.eos.extended_timeout": 200.0,
			}), func([]float32) (float64, error) {
				defer close(predictionReturned)
				close(predictionStarted)
				<-finishPrediction
				return 0.9, nil
			})
			endOfSpeech.turnStopTimeout = 200 * time.Millisecond
			t.Cleanup(func() {
				select {
				case <-finishPrediction:
				default:
					close(finishPrediction)
				}
				select {
				case <-predictionReturned:
				case <-time.After(time.Second):
					t.Error("prediction did not return after being released")
				}
				closeTestEndOfSpeech(endOfSpeech)
			})
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
			if finalBeforePrediction {
				require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("committed transcript", true)))
			}
			stoppedAt := time.Now()
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			select {
			case <-predictionStarted:
			case <-time.After(time.Second):
				t.Fatal("prediction did not start")
			}
			if !finalBeforePrediction {
				require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("committed transcript", true)))
			}
			select {
			case packet := <-completed:
				assert.Equal(t, "committed transcript", packet.Speech)
				assert.GreaterOrEqual(t, time.Since(stoppedAt), 200*time.Millisecond)
				assert.Less(t, time.Since(stoppedAt), 600*time.Millisecond)
			case <-time.After(600 * time.Millisecond):
				t.Fatal("blocked prediction prevented stopped-turn completion")
			}
			select {
			case <-predictionReturned:
				t.Fatal("test predictor returned before release")
			default:
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("next turn", true)))
			close(finishPrediction)
			select {
			case <-predictionReturned:
			case <-time.After(time.Second):
				t.Fatal("stale prediction did not return")
			}
			select {
			case packet := <-completed:
				assert.Equal(t, "next turn", packet.Speech)
			case <-time.After(time.Second):
				t.Fatal("stale prediction blocked the next turn")
			}
			select {
			case packet := <-completed:
				t.Fatalf("stale prediction completed an extra turn: %+v", packet)
			case <-time.After(50 * time.Millisecond):
			}
		})
	}
}

func TestEOS_PredictionCancellation(t *testing.T) {
	for _, action := range []string{"deadline", "audio completion", "speech resumed", "typed text", "close"} {
		t.Run(action, func(t *testing.T) {
			predictionStarted := make(chan struct{})
			predictionCanceled := make(chan error, 1)
			predictionReturned := make(chan struct{})
			completed := make(chan internal_type.EndOfSpeechPacket, 4)
			extendedTimeout := 200.0
			if action == "audio completion" {
				extendedTimeout = 1000.0
			}
			endOfSpeech := newTestEOS(func(ctx context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- packet
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": 50.0,
				"microphone.eos.extended_timeout": extendedTimeout,
			}))
			endOfSpeech.turnStopTimeout = 200 * time.Millisecond
			if action == "audio completion" {
				endOfSpeech.turnStopTimeout = time.Second
			}
			endOfSpeech.predictor = testPredictor{predictContext: func(ctx context.Context, audio []float32) (float64, error) {
				defer close(predictionReturned)
				close(predictionStarted)
				<-ctx.Done()
				predictionCanceled <- ctx.Err()
				return 0, ctx.Err()
			}}
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(func() {
				cancel()
				select {
				case <-predictionReturned:
				case <-time.After(time.Second):
					t.Error("canceled prediction did not return")
				}
				closeTestEndOfSpeech(endOfSpeech)
			})
			require.NoError(t, endOfSpeech.Execute(ctx, internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(ctx, audioInput(1600)))
			require.NoError(t, endOfSpeech.Execute(ctx, sttInput("committed transcript", true)))
			require.NoError(t, endOfSpeech.Execute(ctx, internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			select {
			case <-predictionStarted:
			case <-time.After(time.Second):
				t.Fatal("prediction did not start")
			}
			switch action {
			case "audio completion":
				require.NoError(t, endOfSpeech.Execute(ctx, internal_type.EndOfSpeechAudioPacket{Audio: make([]byte, 32000)}))
			case "speech resumed":
				require.NoError(t, endOfSpeech.Execute(ctx, internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				}))
			case "typed text":
				require.NoError(t, endOfSpeech.Execute(ctx, userInput("typed replacement")))
			case "close":
				closed := make(chan error, 1)
				go func() { closed <- endOfSpeech.Close(ctx) }()
				select {
				case err := <-closed:
					require.NoError(t, err)
				case <-time.After(time.Second):
					t.Fatal("close waited for the prediction deadline")
				}
			}
			cancellationTimeout := time.Second
			if action == "audio completion" {
				cancellationTimeout = 200 * time.Millisecond
			}
			select {
			case err := <-predictionCanceled:
				if action == "deadline" {
					assert.Error(t, err)
				} else {
					assert.ErrorIs(t, err, context.Canceled)
				}
			case <-time.After(cancellationTimeout):
				t.Fatal("prediction was not canceled")
			}
			if action == "deadline" || action == "audio completion" || action == "typed text" {
				select {
				case packet := <-completed:
					if action == "typed text" {
						assert.Equal(t, "typed replacement", packet.Speech)
					} else {
						assert.Equal(t, "committed transcript", packet.Speech)
					}
				case <-time.After(cancellationTimeout):
					t.Fatal("expected turn did not complete after prediction cancellation")
				}
			}
			select {
			case packet := <-completed:
				t.Fatalf("canceled prediction completed an extra turn: %+v", packet)
			case <-time.After(250 * time.Millisecond):
			}
		})
	}
}

func TestEOS_PredictionDoesNotBlockControlDispatch(t *testing.T) {
	predictionStarted := make(chan struct{})
	predictionCanceled := make(chan struct{})
	releasePrediction := make(chan struct{})
	controlDone := make(chan struct{})
	sttEnded := make(chan struct{})
	speechResumed := make(chan struct{})
	var releaseOnce sync.Once
	endOfSpeech := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil }, nil)
	endOfSpeech.predictor = testPredictor{predictContext: func(ctx context.Context, audio []float32) (float64, error) {
		assert.Len(t, audio, 1600)
		close(predictionStarted)
		<-ctx.Done()
		close(predictionCanceled)
		<-releasePrediction
		return 0.9, nil
	}}
	ctx, cancel := context.WithCancel(t.Context())
	channels := adapter_channel.NewRequestorChannels()
	go func() {
		defer close(controlDone)
		channels.RunControl(ctx, func(envelope adapter_channel.Envelope) {
			if _, ok := envelope.Pkt.(internal_type.SpeechToTextEndPacket); ok {
				endOfSpeech.mu.RLock()
				assert.Equal(t, vadStateEnded, endOfSpeech.state.vadState)
				assert.Equal(t, uint64(2), endOfSpeech.state.vadRevision)
				assert.Equal(t, uint64(320), endOfSpeech.state.silenceSamples)
				assert.Equal(t, "committed", endOfSpeech.state.segment.FinalText)
				endOfSpeech.mu.RUnlock()
				close(sttEnded)
				return
			}
			assert.NoError(t, endOfSpeech.Execute(envelope.Ctx, envelope.Pkt))
			if packet, ok := envelope.Pkt.(internal_type.InterruptionDetectedPacket); ok && packet.Event == internal_type.InterruptionEventStart {
				close(speechResumed)
			}
		})
	}()
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(releasePrediction) })
		cancel()
		<-controlDone
		closeTestEndOfSpeech(endOfSpeech)
	})
	require.NoError(t, endOfSpeech.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(ctx, audioInput(1600)))
	channels.OnControl(adapter_channel.Envelope{Ctx: ctx, Pkt: internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}})
	select {
	case <-predictionStarted:
	case <-time.After(time.Second):
		t.Fatal("prediction did not start")
	}
	for _, packet := range []internal_type.Packet{
		audioInput(320), sttInput("committed", true), internal_type.SpeechToTextEndPacket{},
		internal_type.InterruptionDetectedPacket{
			Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		},
	} {
		channels.OnControl(adapter_channel.Envelope{Ctx: ctx, Pkt: packet})
	}
	for _, signal := range []<-chan struct{}{sttEnded, speechResumed, predictionCanceled} {
		select {
		case <-signal:
		case <-time.After(time.Second):
			t.Fatal("prediction blocked ordered control dispatch or resumed-speech cancellation")
		}
	}
	releaseOnce.Do(func() { close(releasePrediction) })
	endOfSpeech.predictorMu.Lock()
	endOfSpeech.predictorMu.Unlock()
	endOfSpeech.mu.RLock()
	defer endOfSpeech.mu.RUnlock()
	assert.Equal(t, vadStateSpeaking, endOfSpeech.state.vadState)
	assert.Equal(t, uint64(3), endOfSpeech.state.vadRevision)
	assert.Equal(t, turnStatePending, endOfSpeech.state.turnState)
	assert.Zero(t, endOfSpeech.state.confidence)
}

func TestEOS_PendingPredictionUsesOrderedAudioSnapshot(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	latestStarted := make(chan []float32, 1)
	releaseLatest := make(chan struct{})
	var calls atomic.Int32
	endOfSpeech := newTestEOSWithPredictor(func(context.Context, ...internal_type.Packet) error { return nil }, nil,
		func(audio []float32) (float64, error) {
			if calls.Add(1) == 1 {
				close(firstStarted)
				<-releaseFirst
				return 0.9, nil
			}
			latestStarted <- audio
			<-releaseLatest
			return 0.1, nil
		})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(releaseFirst) })
		close(releaseLatest)
		closeTestEndOfSpeech(endOfSpeech)
	})
	start := internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart}
	stop := internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd}
	require.NoError(t, endOfSpeech.Execute(t.Context(), start))
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), stop))
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first prediction did not start")
	}
	for range 20 {
		require.NoError(t, endOfSpeech.Execute(t.Context(), start))
		require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(320)))
		require.NoError(t, endOfSpeech.Execute(t.Context(), stop))
	}
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(320)))
	assert.Equal(t, int32(1), calls.Load())
	releaseOnce.Do(func() { close(releaseFirst) })
	select {
	case audio := <-latestStarted:
		assert.Len(t, audio, 1600+20*320)
	case <-time.After(time.Second):
		t.Fatal("latest pending prediction did not start")
	}
	endOfSpeech.mu.RLock()
	assert.Equal(t, turnStatePending, endOfSpeech.state.turnState)
	assert.Zero(t, endOfSpeech.state.confidence)
	assert.Len(t, endOfSpeech.audioBuffer, 1600+21*320)
	endOfSpeech.mu.RUnlock()
	assert.Equal(t, int32(2), calls.Load())
}

func TestEOS_CloseJoinsPredictionBeforeDestroy(t *testing.T) {
	predictionStarted := make(chan struct{})
	predictionCanceled := make(chan struct{})
	releasePrediction := make(chan struct{})
	predictionReturned := make(chan struct{})
	destroyed := make(chan struct{})
	closed := make(chan error, 1)
	var calls atomic.Int32
	var releaseOnce sync.Once
	endOfSpeech := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil }, nil)
	endOfSpeech.predictor = testPredictor{
		predictContext: func(ctx context.Context, audio []float32) (float64, error) {
			calls.Add(1)
			close(predictionStarted)
			<-ctx.Done()
			close(predictionCanceled)
			<-releasePrediction
			close(predictionReturned)
			return 0.9, nil
		},
		destroy: func() {
			select {
			case <-predictionReturned:
			default:
				t.Error("destroy raced active native prediction")
			}
			select {
			case <-endOfSpeech.predictionDone:
			default:
				t.Error("destroy preceded prediction worker shutdown")
			}
			close(destroyed)
		},
	}
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(releasePrediction) })
		closeTestEndOfSpeech(endOfSpeech)
	})
	start := internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart}
	stop := internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd}
	require.NoError(t, endOfSpeech.Execute(t.Context(), start))
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), stop))
	select {
	case <-predictionStarted:
	case <-time.After(time.Second):
		t.Fatal("prediction did not start")
	}
	require.NoError(t, endOfSpeech.Execute(t.Context(), start))
	require.NoError(t, endOfSpeech.Execute(t.Context(), stop))
	go func() { closed <- endOfSpeech.Close(t.Context()) }()
	select {
	case <-endOfSpeech.stopCh:
	case <-time.After(time.Second):
		t.Fatal("close did not stop prediction admission")
	}
	select {
	case <-predictionCanceled:
	case <-time.After(time.Second):
		t.Fatal("prediction was not canceled")
	}
	require.NoError(t, endOfSpeech.Execute(t.Context(), start))
	require.NoError(t, endOfSpeech.Execute(t.Context(), stop))
	select {
	case <-closed:
		t.Fatal("close returned while native prediction was active")
	default:
	}
	releaseOnce.Do(func() { close(releasePrediction) })
	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("close did not join prediction worker")
	}
	select {
	case <-destroyed:
	default:
		t.Fatal("close did not destroy predictor")
	}
	require.NoError(t, endOfSpeech.Close(t.Context()))
	assert.Equal(t, int32(1), calls.Load())
	assert.Nil(t, endOfSpeech.predictor)
	assert.Nil(t, endOfSpeech.cancelPrediction)
}

func TestEOS_CloseDuringPredictionAdmission(t *testing.T) {
	var predictionCalls atomic.Int32
	var destroyCalls atomic.Int32
	endOfSpeech := &pipecatEndOfSpeech{
		onPacket: func(context.Context, ...internal_type.Packet) error { return nil },
		predictor: testPredictor{
			predict: func([]float32) (float64, error) {
				predictionCalls.Add(1)
				return 0.9, nil
			},
			destroy: func() { destroyCalls.Add(1) },
		},
		fallbackTimeout: time.Second,
		turnStopTimeout: time.Second,
		audioBuffer:     []float32{0.1},
		commandCh:       make(chan workerCommand, 1),
		stopCh:          make(chan struct{}),
		state: &endOfSpeechState{
			segment: speechSegment{FinalText: "committed", Text: "committed"},
		},
	}
	endOfSpeech.commandCh <- workerCommand{}
	dispatchDone := make(chan struct{})
	go func() {
		defer close(dispatchDone)
		assert.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
			Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
		}))
	}()
	t.Cleanup(func() {
		closeTestEndOfSpeech(endOfSpeech)
		<-dispatchDone
	})
	require.Eventually(t, func() bool {
		endOfSpeech.mu.RLock()
		defer endOfSpeech.mu.RUnlock()
		return endOfSpeech.state.vadState == vadStateEnded
	}, time.Second, time.Millisecond)
	require.NoError(t, endOfSpeech.Close(t.Context()))
	select {
	case <-dispatchDone:
	case <-time.After(time.Second):
		t.Fatal("close did not release VAD-end dispatch")
	}
	assert.Zero(t, predictionCalls.Load())
	assert.Equal(t, int32(1), destroyCalls.Load())
	assert.Nil(t, endOfSpeech.predictionDone)
	assert.Nil(t, endOfSpeech.cancelPrediction)
}
