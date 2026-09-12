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

type activityCompletion struct {
	packet internal_type.EndOfSpeechPacket
	at     time.Time
}

func TestEOS_TranscriptActivityPostponesTurnStopRecovery(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		firstFinal      bool
		secondChunk     string
		wantSpeech      string
		fallbackTimeout float64
	}{
		{name: "interim update", secondChunk: "there", wantSpeech: "hello there", fallbackTimeout: 80},
		{name: "final update", firstFinal: true, secondChunk: "friend", wantSpeech: "hello there friend", fallbackTimeout: 80},
		{name: "longer transcript safety", secondChunk: "there", wantSpeech: "hello there", fallbackTimeout: 2200},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			completed := make(chan activityCompletion, 4)
			endOfSpeech := newTestEOSWithPredictor(func(_ context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- activityCompletion{packet: packet, at: time.Now()}
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": testCase.fallbackTimeout,
				"microphone.eos.extended_timeout": 200.0,
			}), func([]float32) (float64, error) { return 0.1, nil })
			endOfSpeech.turnStopTimeout = 800 * time.Millisecond
			defer closeTestEndOfSpeech(endOfSpeech)

			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("hello", true)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			endOfSpeech.mu.RLock()
			initialDeadline := endOfSpeech.state.turnStopDeadline
			transcriptDeadline := endOfSpeech.state.transcriptDeadline
			endOfSpeech.mu.RUnlock()

			select {
			case result := <-completed:
				t.Fatalf("incomplete turn completed before transcript activity: %+v", result.packet)
			case <-time.After(time.Until(initialDeadline.Add(-400 * time.Millisecond))):
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("there", testCase.firstFinal)))
			endOfSpeech.mu.RLock()
			activityDeadline := endOfSpeech.state.turnStopDeadline
			endOfSpeech.mu.RUnlock()
			assert.True(t, activityDeadline.After(initialDeadline), "text activity must extend recovery")

			select {
			case result := <-completed:
				t.Fatalf("transcript activity did not postpone the original watchdog: %+v", result.packet)
			case <-time.After(time.Until(initialDeadline.Add(100 * time.Millisecond))):
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput(testCase.secondChunk, true)))
			endOfSpeech.mu.RLock()
			finalDeadline := endOfSpeech.state.turnStopDeadline
			endOfSpeech.mu.RUnlock()
			assert.True(t, finalDeadline.After(activityDeadline), "later committed text must extend recovery again")

			select {
			case result := <-completed:
				t.Fatalf("turn completed before the latest inactivity deadline: %+v", result.packet)
			case <-time.After(time.Until(finalDeadline.Add(-400 * time.Millisecond))):
			}
			for _, script := range []string{"", " \t\n "} {
				for _, final := range []bool{false, true} {
					require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput(script, final)))
				}
			}
			endOfSpeech.mu.RLock()
			deadlineAfterBlanks := endOfSpeech.state.turnStopDeadline
			transcriptDeadlineAfterBlanks := endOfSpeech.state.transcriptDeadline
			endOfSpeech.mu.RUnlock()
			assert.Equal(t, finalDeadline, deadlineAfterBlanks, "blank STT must not extend inactivity")
			assert.Equal(t, transcriptDeadline, transcriptDeadlineAfterBlanks, "blank STT must not change transcript safety")
			recoveryDeadline := finalDeadline
			if transcriptDeadline.After(recoveryDeadline) {
				recoveryDeadline = transcriptDeadline
			}
			select {
			case result := <-completed:
				assert.Equal(t, testCase.wantSpeech, result.packet.Speech)
				assert.False(t, result.at.Before(finalDeadline), "recovery must wait for inactivity after the last final")
				assert.False(t, result.at.Before(transcriptDeadline), "recovery must also wait for transcript safety")
			case <-time.After(time.Until(recoveryDeadline.Add(time.Second))):
				t.Fatal("committed text did not complete after transcript inactivity")
			}
			select {
			case result := <-completed:
				t.Fatalf("turn completed twice: %+v", result.packet)
			case <-time.After(50 * time.Millisecond):
			}
		})
	}
}

func TestEOS_CompleteTurnActivityKeepsTranscriptDeadline(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		probability   float64
		audioComplete bool
	}{
		{name: "model complete", probability: 0.9},
		{name: "audio silence complete", probability: 0.1, audioComplete: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			completed := make(chan activityCompletion, 4)
			endOfSpeech := newTestEOSWithPredictor(func(_ context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- activityCompletion{packet: packet, at: time.Now()}
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": 600.0,
				"microphone.eos.extended_timeout": 200.0,
			}), func([]float32) (float64, error) { return testCase.probability, nil })
			endOfSpeech.turnStopTimeout = 3 * time.Second
			defer closeTestEndOfSpeech(endOfSpeech)

			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("hello", true)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			endOfSpeech.mu.RLock()
			transcriptDeadline := endOfSpeech.state.transcriptDeadline
			initialTurnStopDeadline := endOfSpeech.state.turnStopDeadline
			endOfSpeech.mu.RUnlock()

			for index, final := range []bool{false, true} {
				updateAt := transcriptDeadline.Add(time.Duration(index-2) * 200 * time.Millisecond)
				select {
				case result := <-completed:
					t.Fatalf("turn completed before transcript safety elapsed: %+v", result.packet)
				case <-time.After(time.Until(updateAt)):
				}
				require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("there", final)))
			}
			endOfSpeech.mu.RLock()
			deadlineAfterActivity := endOfSpeech.state.transcriptDeadline
			turnStopDeadline := endOfSpeech.state.turnStopDeadline
			endOfSpeech.mu.RUnlock()
			assert.Equal(t, transcriptDeadline, deadlineAfterActivity)
			assert.True(t, turnStopDeadline.After(initialTurnStopDeadline))
			if testCase.audioComplete {
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.EndOfSpeechAudioPacket{
					Audio: make([]byte, 6400),
				}))
			}

			select {
			case result := <-completed:
				assert.Equal(t, "hello there", result.packet.Speech)
				assert.False(t, result.at.Before(transcriptDeadline), "COMPLETE must retain transcript safety")
				assert.True(t, result.at.Before(turnStopDeadline), "COMPLETE must not wait for recovery")
			case <-time.After(time.Until(transcriptDeadline.Add(time.Second))):
				t.Fatal("COMPLETE was delayed by the transcript inactivity watchdog")
			}
		})
	}
}

func TestEOS_TranscriptActivityDoesNotExtendPredictionCancellation(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		fallbackTimeout float64
	}{
		{name: "watchdog sets prediction budget", fallbackTimeout: 80},
		{name: "transcript safety sets prediction budget", fallbackTimeout: 1200},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			completed := make(chan activityCompletion, 4)
			predictionStarted := make(chan time.Time, 1)
			predictionCanceled := make(chan struct {
				at  time.Time
				err error
			}, 1)
			predictionDone := make(chan struct{})
			endOfSpeech := newTestEOS(func(_ context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- activityCompletion{packet: packet, at: time.Now()}
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": testCase.fallbackTimeout,
				"microphone.eos.extended_timeout": 200.0,
			}))
			endOfSpeech.turnStopTimeout = time.Second
			endOfSpeech.predictor = testPredictor{predictContext: func(ctx context.Context, _ []float32) (float64, error) {
				defer close(predictionDone)
				deadline, _ := ctx.Deadline()
				predictionStarted <- deadline
				<-ctx.Done()
				predictionCanceled <- struct {
					at  time.Time
					err error
				}{at: time.Now(), err: ctx.Err()}
				return 0, ctx.Err()
			}}
			ctx, cancel := context.WithCancel(t.Context())
			defer closeTestEndOfSpeech(endOfSpeech)
			defer cancel()
			require.NoError(t, endOfSpeech.Execute(ctx, internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(ctx, audioInput(1600)))
			require.NoError(t, endOfSpeech.Execute(ctx, sttInput("hello", true)))
			require.NoError(t, endOfSpeech.Execute(ctx, internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			t.Cleanup(func() {
				select {
				case <-predictionDone:
				case <-time.After(time.Second):
					t.Error("canceled prediction did not return")
				}
			})

			var predictionDeadline time.Time
			select {
			case predictionDeadline = <-predictionStarted:
				require.False(t, predictionDeadline.IsZero(), "inference must have a fixed deadline")
			case <-time.After(time.Second):
				t.Fatal("prediction did not start")
			}
			endOfSpeech.mu.RLock()
			expectedPredictionDeadline := endOfSpeech.state.turnStopDeadline
			if endOfSpeech.state.transcriptDeadline.After(expectedPredictionDeadline) {
				expectedPredictionDeadline = endOfSpeech.state.transcriptDeadline
			}
			endOfSpeech.mu.RUnlock()
			assert.Equal(t, expectedPredictionDeadline, predictionDeadline)
			select {
			case result := <-completed:
				t.Fatalf("blocked prediction completed before transcript activity: %+v", result.packet)
			case <-time.After(time.Until(predictionDeadline.Add(-400 * time.Millisecond))):
			}
			require.NoError(t, endOfSpeech.Execute(ctx, sttInput("there", false)))
			endOfSpeech.mu.RLock()
			activityDeadline := endOfSpeech.state.turnStopDeadline
			endOfSpeech.mu.RUnlock()
			assert.True(t, activityDeadline.After(predictionDeadline))

			select {
			case result := <-predictionCanceled:
				assert.ErrorIs(t, result.err, context.DeadlineExceeded)
				assert.False(t, result.at.Before(predictionDeadline), "inference canceled before its initial deadline")
				assert.True(t, result.at.Before(activityDeadline), "STT activity must not extend inference")
			case <-time.After(time.Until(predictionDeadline.Add(300 * time.Millisecond))):
				t.Fatal("prediction did not cancel at its original deadline")
			}
			select {
			case <-predictionDone:
			case <-time.After(300 * time.Millisecond):
				t.Fatal("canceled native prediction did not return")
			}
			select {
			case result := <-completed:
				t.Fatalf("inference cancellation caused premature EOS: %+v", result.packet)
			case <-time.After(100 * time.Millisecond):
			}
			require.NoError(t, endOfSpeech.Execute(ctx, sttInput("there friend", true)))
			endOfSpeech.mu.RLock()
			finalDeadline := endOfSpeech.state.turnStopDeadline
			endOfSpeech.mu.RUnlock()
			assert.True(t, finalDeadline.After(activityDeadline))

			select {
			case result := <-completed:
				assert.Equal(t, "hello there friend", result.packet.Speech)
				assert.False(t, result.at.Before(finalDeadline), "late final must receive a full inactivity budget")
			case <-time.After(time.Until(finalDeadline.Add(time.Second))):
				t.Fatal("late final did not complete after transcript inactivity")
			}
		})
	}
}

func TestEOS_InterimOnlyActivityNeverCompletesTurn(t *testing.T) {
	for _, probability := range []float64{0.1, 0.9} {
		name := "incomplete"
		if probability > defaultPctThreshold {
			name = "complete"
		}
		t.Run(name, func(t *testing.T) {
			completed := make(chan internal_type.EndOfSpeechPacket, 4)
			endOfSpeech := newTestEOSWithPredictor(func(_ context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- packet
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": 80.0,
				"microphone.eos.extended_timeout": 100.0,
			}), func([]float32) (float64, error) { return probability, nil })
			endOfSpeech.turnStopTimeout = 400 * time.Millisecond
			defer closeTestEndOfSpeech(endOfSpeech)
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("hello", false)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			select {
			case packet := <-completed:
				t.Fatalf("interim-only text completed before later activity: %+v", packet)
			case <-time.After(200 * time.Millisecond):
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("hello there", false)))
			endOfSpeech.mu.RLock()
			activityDeadline := endOfSpeech.state.turnStopDeadline
			endOfSpeech.mu.RUnlock()
			select {
			case packet := <-completed:
				t.Fatalf("interim-only text completed after inactivity: %+v", packet)
			case <-time.After(time.Until(activityDeadline.Add(150 * time.Millisecond))):
			}
		})
	}
}
