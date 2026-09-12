// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/require"
)

func TestLivekitEndOfSpeech_AudioCountsCompletePCM16Samples(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		audio           []byte
		expectedSamples uint64
	}{
		{name: "empty"},
		{name: "incomplete sample", audio: []byte{0}},
		{name: "complete samples", audio: []byte{0, 0, 255, 127}, expectedSamples: 2},
		{name: "odd trailing byte", audio: []byte{0, 128, 0}, expectedSamples: 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			endOfSpeech := &livekitEndOfSpeech{}
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.EndOfSpeechAudioPacket{Audio: testCase.audio}))
			require.Equal(t, testCase.expectedSamples, endOfSpeech.audioSamples)
		})
	}
}

func TestLivekitEndOfSpeech_PredictionDoesNotBlockPackets(t *testing.T) {
	for _, useVAD := range []bool{false, true} {
		name := "final transcript"
		if useVAD {
			name = "VAD end"
		}
		t.Run(name, func(t *testing.T) {
			predictionStarted := make(chan struct{}, 1)
			predictionCanceled := make(chan struct{}, 1)
			completed := make(chan internal_type.EndOfSpeechPacket, 1)
			endOfSpeech := &livekitEndOfSpeech{
				onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
							completed <- packet
						}
					}
					return nil
				},
				predictor: testPredictor{predictContext: func(ctx context.Context, text string) (float64, error) {
					predictionStarted <- struct{}{}
					<-ctx.Done()
					predictionCanceled <- struct{}{}
					return 0, ctx.Err()
				}},
				threshold: 0.5, quickTimeout: time.Millisecond, silenceTimeout: time.Second,
				maxHistory: int(defaultMaxHistory), commandCh: make(chan struct{}, 1),
				stopCh: make(chan struct{}), workerDone: make(chan struct{}), state: &endOfSpeechState{},
			}
			go endOfSpeech.worker()
			defer endOfSpeech.Close(context.Background())
			if useVAD {
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				}))
			}
			executed := make(chan error, 1)
			go func() {
				err := endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "not finished"})
				if err == nil && useVAD {
					err = endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
						Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
					})
				}
				executed <- err
			}()
			select {
			case err := <-executed:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("packet handling blocked on inference")
			}
			select {
			case <-predictionStarted:
			case <-time.After(time.Second):
				t.Fatal("prediction did not start")
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			select {
			case <-predictionCanceled:
			case <-time.After(time.Second):
				t.Fatal("resumed speech did not cancel inference")
			}
			require.NoError(t, endOfSpeech.Close(context.Background()))
			require.Empty(t, completed)
		})
	}
}

func TestLivekitEndOfSpeech_NewFinalCancelsPrediction(t *testing.T) {
	predicted := make(chan string, 2)
	completed := make(chan internal_type.EndOfSpeechPacket, 2)
	var predictionCount atomic.Int32
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					completed <- packet
				}
			}
			return nil
		},
		predictor: testPredictor{predictContext: func(ctx context.Context, text string) (float64, error) {
			predicted <- text
			if predictionCount.Add(1) == 1 {
				<-ctx.Done()
				return 0, ctx.Err()
			}
			return 0.9, nil
		}},
		threshold: 0.5, quickTimeout: time.Millisecond, silenceTimeout: time.Second,
		maxHistory: int(defaultMaxHistory), commandCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}), workerDone: make(chan struct{}), state: &endOfSpeechState{},
	}
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "I wanted"}))
	select {
	case text := <-predicted:
		require.Contains(t, text, "I wanted")
	case <-time.After(time.Second):
		t.Fatal("first prediction did not start")
	}
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "to talk"}))
	select {
	case packet := <-completed:
		require.Equal(t, "I wanted to talk", packet.Speech)
	case <-time.After(time.Second):
		t.Fatal("latest transcript did not replace canceled inference")
	}
	require.Contains(t, <-predicted, "I wanted to talk")
	require.NoError(t, endOfSpeech.Close(context.Background()))
	require.Empty(t, completed)
}

func TestLivekitEndOfSpeech_PredictionDeadlineUsesMinimumDelay(t *testing.T) {
	completed := make(chan internal_type.EndOfSpeechPacket, 1)
	predictionError := make(chan error, 1)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					completed <- packet
				}
			}
			return nil
		},
		predictor: testPredictor{predictContext: func(ctx context.Context, text string) (float64, error) {
			<-ctx.Done()
			predictionError <- ctx.Err()
			return 0, ctx.Err()
		}},
		threshold: 0.5, quickTimeout: 10 * time.Millisecond, silenceTimeout: 10 * time.Second,
		maxHistory: int(defaultMaxHistory), commandCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}), workerDone: make(chan struct{}), state: &endOfSpeechState{},
	}
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())
	started := time.Now()
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "finished"}))
	select {
	case packet := <-completed:
		require.Equal(t, "finished", packet.Speech)
		require.GreaterOrEqual(t, time.Since(started), predictionTimeout)
		require.ErrorIs(t, <-predictionError, context.DeadlineExceeded)
	case <-time.After(predictionTimeout + time.Second):
		t.Fatal("inference deadline did not fall back to the elapsed minimum delay")
	}
}

func TestLivekitEndOfSpeech_InferenceConsumesEndpointDelay(t *testing.T) {
	for _, probability := range []float64{0.1, 0.9} {
		name := "maximum delay"
		if probability >= 0.5 {
			name = "minimum delay"
		}
		t.Run(name, func(t *testing.T) {
			predictionStarted := make(chan struct{})
			releasePrediction := make(chan struct{})
			completed := make(chan internal_type.EndOfSpeechPacket, 1)
			endOfSpeech := &livekitEndOfSpeech{
				onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
							completed <- packet
						}
					}
					return nil
				},
				predictor: testPredictor{predictContext: func(ctx context.Context, text string) (float64, error) {
					close(predictionStarted)
					select {
					case <-releasePrediction:
						return probability, nil
					case <-ctx.Done():
						return 0, ctx.Err()
					}
				}},
				threshold: 0.5, quickTimeout: 200 * time.Millisecond, silenceTimeout: 250 * time.Millisecond,
				maxHistory: int(defaultMaxHistory), commandCh: make(chan struct{}, 1),
				stopCh: make(chan struct{}), workerDone: make(chan struct{}), state: &endOfSpeechState{},
			}
			go endOfSpeech.worker()
			defer endOfSpeech.Close(context.Background())
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "finished"}))
			select {
			case <-predictionStarted:
			case <-time.After(time.Second):
				t.Fatal("prediction did not start")
			}
			time.Sleep(endOfSpeech.silenceTimeout)
			close(releasePrediction)
			select {
			case packet := <-completed:
				require.Equal(t, "finished", packet.Speech)
			case <-time.After(150 * time.Millisecond):
				t.Fatal("inference added another full endpoint delay")
			}
		})
	}
}

func TestLivekitEndOfSpeech_CloseJoinsCallback(t *testing.T) {
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	closedEvent := make(chan struct{}, 1)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				switch packet := packet.(type) {
				case internal_type.EndOfSpeechPacket:
					close(callbackStarted)
					<-releaseCallback
				case internal_type.ObservabilityEventRecordPacket:
					if packet.Record.Event == observability.EOSClosed {
						closedEvent <- struct{}{}
					}
				}
			}
			return nil
		},
		commandCh: make(chan struct{}, 1), stopCh: make(chan struct{}),
		workerDone: make(chan struct{}), state: &endOfSpeechState{},
	}
	go endOfSpeech.worker()
	t.Cleanup(func() { require.NoError(t, endOfSpeech.Close(context.Background())) })
	t.Cleanup(func() { close(releaseCallback) })
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.UserTextReceivedPacket{Text: "first"}))
	select {
	case <-callbackStarted:
	case <-time.After(time.Second):
		t.Fatal("completion callback did not start")
	}
	closeContext, cancelClose := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancelClose()
	require.ErrorIs(t, endOfSpeech.Close(closeContext), context.DeadlineExceeded)
	require.Empty(t, closedEvent, "closed event must follow completion callback exit")
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.UserTextReceivedPacket{Text: "ignored after close"}))
	require.Empty(t, endOfSpeech.commandCh)
}

func TestLivekitEndOfSpeech_InterimDeliveryPrecedesExpiredFinal(t *testing.T) {
	interimStarted := make(chan struct{})
	releaseInterim := make(chan struct{})
	delivered := make(chan string, 4)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				switch packet := packet.(type) {
				case internal_type.InterimEndOfSpeechPacket:
					if packet.Speech == "committed preview" {
						close(interimStarted)
						<-releaseInterim
						delivered <- "interim"
					}
				case internal_type.EndOfSpeechPacket:
					delivered <- packet.Speech
				}
			}
			return nil
		},
		predictor: testPredictor{predict: func(string) (float64, error) { return 0.9, nil }},
		threshold: 0.5, quickTimeout: 50 * time.Millisecond, silenceTimeout: time.Second,
		maxHistory: int(defaultMaxHistory), commandCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}), workerDone: make(chan struct{}), state: &endOfSpeechState{},
	}
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "committed"}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "preview", Interim: true}))
	select {
	case <-interimStarted:
	case <-time.After(time.Second):
		close(releaseInterim)
		t.Fatal("interim callback did not start")
	}
	time.Sleep(2 * endOfSpeech.quickTimeout)
	if len(delivered) != 0 {
		close(releaseInterim)
		t.Fatal("final delivery overtook an admitted preview")
	}
	close(releaseInterim)
	for _, expected := range []string{"interim", "committed"} {
		select {
		case actual := <-delivered:
			require.Equal(t, expected, actual)
		case <-time.After(time.Second):
			t.Fatalf("missing delivery %q", expected)
		}
	}
}

func TestLivekitEndOfSpeech_CanceledCallerCannotComplete(t *testing.T) {
	predictionStarted := make(chan struct{})
	completed := make(chan internal_type.EndOfSpeechPacket, 1)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if packet, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					completed <- packet
				}
			}
			return nil
		},
		predictor: testPredictor{predictContext: func(ctx context.Context, text string) (float64, error) {
			close(predictionStarted)
			<-ctx.Done()
			return 0, ctx.Err()
		}},
		quickTimeout: time.Millisecond, maxHistory: int(defaultMaxHistory),
		commandCh: make(chan struct{}, 1), stopCh: make(chan struct{}),
		workerDone: make(chan struct{}), state: &endOfSpeechState{},
	}
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, endOfSpeech.Execute(ctx, internal_type.SpeechToTextPacket{Script: "canceled"}))
	select {
	case <-predictionStarted:
	case <-time.After(time.Second):
		t.Fatal("prediction did not start")
	}
	cancel()
	require.ErrorIs(t, endOfSpeech.Execute(ctx, internal_type.SpeechToTextPacket{Script: "ignored"}), context.Canceled)
	require.NoError(t, endOfSpeech.Close(context.Background()))
	require.Empty(t, completed)
}

func TestLivekitEndOfSpeech_IncompletePredictionExcludesInterim(t *testing.T) {
	completed := make(chan internal_type.EndOfSpeechPacket, 4)
	predicted := make(chan string, 4)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					completed <- result
				}
			}
			return nil
		},
		predictor: testPredictor{predict: func(text string) (float64, error) {
			predicted <- text
			return 0.1, nil
		}},
		threshold:      0.5,
		quickTimeout:   10 * time.Millisecond,
		silenceTimeout: 150 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{},
	}
	endOfSpeech.workerDone = make(chan struct{})
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "committed"}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "unconfirmed tail", Interim: true}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}))
	require.Contains(t, <-predicted, "committed")
	select {
	case packet := <-completed:
		t.Fatalf("incomplete prediction completed at minimum delay: %q", packet.Speech)
	case <-time.After(60 * time.Millisecond):
	}
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "another preview", Interim: true}))
	select {
	case packet := <-completed:
		require.Equal(t, "committed", packet.Speech)
		require.Len(t, packet.Speechs, 3)
	case <-time.After(time.Second):
		t.Fatal("committed transcript did not complete at maximum delay")
	}
	require.Empty(t, predicted, "interim transcripts must not invoke prediction")
}

func TestLivekitEndOfSpeech_CallbackPreservesNextTranscript(t *testing.T) {
	completed := make(chan internal_type.EndOfSpeechPacket, 4)
	var endOfSpeech *livekitEndOfSpeech
	endOfSpeech = &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					completed <- result
					if result.Speech == "first" {
						return endOfSpeech.Execute(ctx, internal_type.SpeechToTextPacket{Script: "second"})
					}
				}
			}
			return nil
		},
		predictor:      testPredictor{predict: func(string) (float64, error) { return 0.9, nil }},
		threshold:      0.5,
		quickTimeout:   10 * time.Millisecond,
		silenceTimeout: 500 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{},
	}
	endOfSpeech.workerDone = make(chan struct{})
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.UserTextReceivedPacket{Text: "first"}))
	for _, expected := range []string{"first", "second"} {
		select {
		case packet := <-completed:
			require.Equal(t, expected, packet.Speech)
		case <-time.After(time.Second):
			t.Fatalf("turn %q was lost during completion callback", expected)
		}
	}
	endOfSpeech.mu.RLock()
	history := append([]chatMessage(nil), endOfSpeech.history...)
	endOfSpeech.mu.RUnlock()
	require.Equal(t, []chatMessage{{Role: "user", Content: "first"}, {Role: "user", Content: "second"}}, history)
}

func TestLivekitEndOfSpeech_QueuedTextDoesNotConsumeNewSpeech(t *testing.T) {
	for _, useVAD := range []bool{false, true} {
		name := "without VAD"
		if useVAD {
			name = "with VAD"
		}
		t.Run(name, func(t *testing.T) {
			completed := make(chan internal_type.EndOfSpeechPacket, 4)
			endOfSpeech := &livekitEndOfSpeech{
				onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
							completed <- result
						}
					}
					return nil
				},
				predictor:      testPredictor{predict: func(string) (float64, error) { return 0.9, nil }},
				threshold:      0.5,
				quickTimeout:   10 * time.Millisecond,
				silenceTimeout: time.Second,
				maxHistory:     int(defaultMaxHistory),
				commandCh:      make(chan struct{}, 1),
				stopCh:         make(chan struct{}),
				state:          &endOfSpeechState{},
			}
			defer endOfSpeech.Close(context.Background())
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.UserTextReceivedPacket{Text: "typed"}))
			if useVAD {
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				}))
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "spoken"}))
			if useVAD {
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
				}))
			}
			endOfSpeech.workerDone = make(chan struct{})
			go endOfSpeech.worker()
			for _, expected := range []string{"typed", "spoken"} {
				select {
				case packet := <-completed:
					require.Equal(t, expected, packet.Speech)
				case <-time.After(time.Second):
					t.Fatalf("queued turn %q was lost", expected)
				}
			}
			select {
			case packet := <-completed:
				t.Fatalf("duplicate completion: %+v", packet)
			case <-time.After(30 * time.Millisecond):
			}
		})
	}
}

func TestLivekitEndOfSpeech_DeadlineIncludesVADSilenceAndInference(t *testing.T) {
	for _, audioEnd := range []float64{0, 0.8, 2} {
		t.Run(time.Duration(audioEnd*float64(time.Second)).String(), func(t *testing.T) {
			endOfSpeech := &livekitEndOfSpeech{
				onPacket: func(context.Context, ...internal_type.Packet) error { return nil },
				predictor: testPredictor{predict: func(string) (float64, error) {
					time.Sleep(25 * time.Millisecond)
					return 0.9, nil
				}},
				threshold:      0.5,
				quickTimeout:   100 * time.Millisecond,
				silenceTimeout: time.Second,
				maxHistory:     int(defaultMaxHistory),
				commandCh:      make(chan struct{}, 1),
				stopCh:         make(chan struct{}),
				state:          &endOfSpeechState{},
			}
			defer endOfSpeech.Close(context.Background())
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.EndOfSpeechAudioPacket{Audio: make([]byte, 32000)}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "finished"}))
			beforeEnd := time.Now()
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd, EndAt: audioEnd,
			}))
			require.Len(t, endOfSpeech.commands, 2)
			command := endOfSpeech.commands[1]
			expected := beforeEnd
			if audioEnd > 0 && audioEnd <= 1 {
				expected = expected.Add(-time.Duration((1 - audioEnd) * float64(time.Second)))
			}
			require.WithinDuration(t, expected, command.deadline, 20*time.Millisecond)
			require.Equal(t, endOfSpeech.state.lastSpeechEnd, command.deadline)
			require.True(t, command.predict)
		})
	}
}

func TestLivekitEndOfSpeech_LateFinalKeepsSpeechEndDeadline(t *testing.T) {
	predicted := make(chan string, 4)
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(context.Context, ...internal_type.Packet) error { return nil },
		predictor: testPredictor{predict: func(text string) (float64, error) {
			predicted <- text
			return 0.1, nil
		}},
		threshold:      0.5,
		quickTimeout:   10 * time.Millisecond,
		silenceTimeout: time.Second,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{},
	}
	defer endOfSpeech.Close(context.Background())
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "I wanted"}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}))
	require.Len(t, endOfSpeech.commands, 2)
	first := endOfSpeech.commands[1]
	require.True(t, first.predict)
	require.Equal(t, "I wanted", first.segment.Text)
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "preview", Interim: true}))
	require.Empty(t, predicted)
	require.Len(t, endOfSpeech.commands, 3)
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "to talk"}))
	require.Len(t, endOfSpeech.commands, 4)
	second := endOfSpeech.commands[3]
	require.Equal(t, first.deadline, second.deadline)
	require.Greater(t, second.segment.Revision, first.segment.Revision)
	require.Equal(t, "I wanted to talk", second.segment.Text)
	require.True(t, second.predict)
	require.Empty(t, predicted, "packet handling must only schedule inference")
}

func TestLivekitEndOfSpeech_StaleInferenceCannotEndResumedSpeech(t *testing.T) {
	completed := make(chan internal_type.EndOfSpeechPacket, 4)
	inferenceStarted := make(chan struct{}, 1)
	releaseInference := make(chan struct{})
	endOfSpeech := &livekitEndOfSpeech{
		onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
					completed <- result
				}
			}
			return nil
		},
		predictor: testPredictor{predict: func(string) (float64, error) {
			inferenceStarted <- struct{}{}
			<-releaseInference
			return 0.9, nil
		}},
		threshold:      0.5,
		quickTimeout:   10 * time.Millisecond,
		silenceTimeout: 500 * time.Millisecond,
		maxHistory:     int(defaultMaxHistory),
		commandCh:      make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{},
	}
	endOfSpeech.workerDone = make(chan struct{})
	go endOfSpeech.worker()
	defer endOfSpeech.Close(context.Background())
	executed := make(chan error, 1)
	go func() {
		executed <- endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "first"})
	}()
	<-inferenceStarted
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}))
	close(releaseInference)
	require.NoError(t, <-executed)
	select {
	case packet := <-completed:
		t.Fatalf("stale inference completed resumed speech: %q", packet.Speech)
	case <-time.After(60 * time.Millisecond):
	}
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "second"}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}))
	select {
	case packet := <-completed:
		require.Equal(t, "first second", packet.Speech)
	case <-time.After(time.Second):
		t.Fatal("resumed speech never completed")
	}
}
