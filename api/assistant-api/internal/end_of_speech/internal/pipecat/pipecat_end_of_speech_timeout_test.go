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

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEOS_IncompleteTurnCompletesWithoutMoreAudio(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		audioSamples  int
		probability   float64
		predictionErr error
	}{
		{name: "below threshold", audioSamples: 1600, probability: 0.1},
		{name: "at threshold", audioSamples: 1600, probability: defaultPctThreshold},
		{name: "inference failure", audioSamples: 1600, predictionErr: errPipecatDetectorRunInference},
		{name: "empty audio"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			completed := make(chan internal_type.EndOfSpeechPacket, 4)
			fallbackLogs := make(chan internal_type.ObservabilityLogRecordPacket, 4)
			predictionErrors := make(chan internal_type.ObservabilityLogRecordPacket, 1)
			endOfSpeech := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- packet
					}
					if packet, ok := packet.(internal_type.ObservabilityLogRecordPacket); ok && packet.Record.Level == observability.LevelInfo {
						fallbackLogs <- packet
					}
					if packet, ok := packet.(internal_type.ObservabilityLogRecordPacket); ok && packet.Record.Level == observability.LevelError {
						assert.NoError(t, ctx.Err(), "inference cancellation must not cancel the error callback context")
						predictionErrors <- packet
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": 300.0,
				"microphone.eos.extended_timeout": 400.0,
			}), func([]float32) (float64, error) {
				return testCase.probability, testCase.predictionErr
			})
			endOfSpeech.turnStopTimeout = 400 * time.Millisecond
			defer closeTestEndOfSpeech(endOfSpeech)

			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(testCase.audioSamples)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("committed transcript", true)))
			stoppedAt := time.Now()
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			if testCase.predictionErr != nil {
				select {
				case packet := <-predictionErrors:
					assert.Contains(t, packet.Record.Message, testCase.predictionErr.Error())
				case <-time.After(time.Second):
					t.Fatal("asynchronous inference failure was not reported")
				}
			}

			select {
			case packet := <-completed:
				assert.Equal(t, "committed transcript", packet.Speech)
				assert.GreaterOrEqual(t, time.Since(stoppedAt), 400*time.Millisecond)
				assert.Less(t, time.Since(stoppedAt), 600*time.Millisecond)
			case <-time.After(600 * time.Millisecond):
				t.Fatal("committed transcript remained blocked without additional audio")
			}
			select {
			case packet := <-fallbackLogs:
				assert.Contains(t, packet.Record.Message, "inactivity watchdog")
			default:
				t.Fatal("inactivity watchdog completion was not reported")
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(4800)))
			select {
			case packet := <-completed:
				t.Fatalf("turn completed twice: %+v", packet)
			case <-time.After(30 * time.Millisecond):
			}
		})
	}
}

func TestEOS_IncompleteTurnLateTranscriptResetsWatchdog(t *testing.T) {
	for _, finalBeforeStop := range []bool{false, true} {
		name := "interim only until after deadline"
		if finalBeforeStop {
			name = "committed prefix with later transcript"
		}
		t.Run(name, func(t *testing.T) {
			completed := make(chan internal_type.EndOfSpeechPacket, 4)
			endOfSpeech := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- packet
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": 20.0,
				"microphone.eos.extended_timeout": 400.0,
			}), func([]float32) (float64, error) { return 0.1, nil })
			endOfSpeech.turnStopTimeout = 400 * time.Millisecond
			defer closeTestEndOfSpeech(endOfSpeech)
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("hello", finalBeforeStop)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			endOfSpeech.mu.RLock()
			turnStopDeadline := endOfSpeech.state.turnStopDeadline
			endOfSpeech.mu.RUnlock()
			if finalBeforeStop {
				time.Sleep(250 * time.Millisecond)
			} else {
				select {
				case packet := <-completed:
					t.Fatalf("interim-only transcript completed: %+v", packet)
				case <-time.After(450 * time.Millisecond):
				}
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("pending suffix", false)))
			endOfSpeech.mu.RLock()
			assert.True(t, endOfSpeech.state.turnStopDeadline.After(turnStopDeadline))
			endOfSpeech.mu.RUnlock()
			transcribedAt := time.Now()
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("world", true)))
			select {
			case packet := <-completed:
				if finalBeforeStop {
					assert.Equal(t, "hello world", packet.Speech)
				} else {
					assert.Equal(t, "world", packet.Speech)
				}
				assert.GreaterOrEqual(t, time.Since(transcribedAt), 400*time.Millisecond)
			case <-time.After(600 * time.Millisecond):
				t.Fatal("late committed transcript remained blocked")
			}
		})
	}
}

func TestEOS_IncompleteTurnWatchdogCanceledBySpeechOrClose(t *testing.T) {
	for _, closeBeforeDeadline := range []bool{false, true} {
		name := "speech resumed"
		if closeBeforeDeadline {
			name = "closed"
		}
		t.Run(name, func(t *testing.T) {
			completed := make(chan internal_type.EndOfSpeechPacket, 4)
			endOfSpeech := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- packet
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": 20.0,
				"microphone.eos.extended_timeout": 80.0,
			}), func([]float32) (float64, error) { return 0.1, nil })
			endOfSpeech.turnStopTimeout = 80 * time.Millisecond
			defer closeTestEndOfSpeech(endOfSpeech)
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("hello", true)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			if closeBeforeDeadline {
				require.NoError(t, endOfSpeech.Close(t.Context()))
			} else {
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				}))
			}
			select {
			case packet := <-completed:
				t.Fatalf("invalidated inactivity watchdog completed a turn: %+v", packet)
			case <-time.After(120 * time.Millisecond):
			}
			if closeBeforeDeadline {
				return
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("world", true)))
			stoppedAt := time.Now()
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			select {
			case packet := <-completed:
				assert.Equal(t, "hello world", packet.Speech)
				assert.GreaterOrEqual(t, time.Since(stoppedAt), 80*time.Millisecond)
			case <-time.After(time.Second):
				t.Fatal("resumed speech did not complete after its new inactivity deadline")
			}
		})
	}
}

func TestEOS_IncompleteTurnHonorsLongerTranscriptDeadline(t *testing.T) {
	completed := make(chan internal_type.EndOfSpeechPacket, 4)
	endOfSpeech := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
				completed <- packet
			}
		}
		return nil
	}, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 400.0,
		"microphone.eos.extended_timeout": 300.0,
	}), func([]float32) (float64, error) { return 0.1, nil })
	endOfSpeech.turnStopTimeout = 300 * time.Millisecond
	defer closeTestEndOfSpeech(endOfSpeech)
	for _, packet := range []internal_type.Packet{
		internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart},
		audioInput(1600), sttInput("committed", true),
	} {
		require.NoError(t, endOfSpeech.Execute(t.Context(), packet))
	}
	stoppedAt := time.Now()
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}))
	select {
	case packet := <-completed:
		assert.Equal(t, "committed", packet.Speech)
		assert.GreaterOrEqual(t, time.Since(stoppedAt), 400*time.Millisecond)
		assert.Less(t, time.Since(stoppedAt), 600*time.Millisecond)
	case <-time.After(600 * time.Millisecond):
		t.Fatal("turn did not complete after both deadlines elapsed")
	}
}
