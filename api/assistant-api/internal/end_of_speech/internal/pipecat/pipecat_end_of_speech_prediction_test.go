// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import (
	"context"
	"testing"
	"time"

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
			var predictionErr error
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
					assert.NoError(t, predictionErr)
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
			go func() {
				predictionErr = endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
				})
				close(predictionReturned)
			}()
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
				t.Fatalf("test predictor returned before release: %v", predictionErr)
			default:
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("next turn", true)))
			close(finishPrediction)
			select {
			case <-predictionReturned:
				require.NoError(t, predictionErr)
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
			go func() {
				_ = endOfSpeech.Execute(ctx, internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
				})
				close(predictionReturned)
			}()
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
