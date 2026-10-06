// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"context"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPredictionErrorsRetainCause(t *testing.T) {
	endOfSpeech := &livekitEndOfSpeech{}
	probability, err := endOfSpeech.predictEOU(t.Context(), "")
	require.ErrorIs(t, err, errTurnDetectorEmptyTokenSequence)
	assert.Zero(t, probability)
	probability, err = endOfSpeech.predictEOU(t.Context(), "committed text")
	require.ErrorIs(t, err, errTurnDetectorNil)
	assert.Zero(t, probability)
	endOfSpeech.predictor = testPredictor{predict: func(string) (float64, error) {
		return 0, errTurnDetectorCreateInputIDsTensor
	}}
	probability, err = endOfSpeech.predictEOU(t.Context(), "committed text")
	require.ErrorIs(t, err, errTurnDetectorRunInference)
	require.ErrorIs(t, err, errTurnDetectorCreateInputIDsTensor)
	assert.Zero(t, probability)
}

func TestPredictionFailureReportingKeepsMinimumDelay(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		predictor   turnPredictor
		loggedError error
	}{
		{name: "native failure", predictor: testPredictor{predict: func(string) (float64, error) {
			return 0, errTurnDetectorCreateInputIDsTensor
		}}, loggedError: errTurnDetectorCreateInputIDsTensor},
		{name: "missing detector", loggedError: errTurnDetectorNil},
		{name: "canceled prediction", predictor: testPredictor{predict: func(string) (float64, error) {
			return 0, context.Canceled
		}}},
		{name: "expired prediction", predictor: testPredictor{predict: func(string) (float64, error) {
			return 0, context.DeadlineExceeded
		}}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			completed := make(chan internal_type.EndOfSpeechPacket, 1)
			failures := make(chan internal_type.ObservabilityLogRecordPacket, 2)
			endOfSpeech := &livekitEndOfSpeech{
				onPacket: func(_ context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						switch packet := packet.(type) {
						case internal_type.EndOfSpeechPacket:
							completed <- packet
						case internal_type.ObservabilityLogRecordPacket:
							if packet.Record.Level == observability.LevelError {
								failures <- packet
							}
						}
					}
					return nil
				},
				predictor: testCase.predictor,
				threshold: 0.5, quickTimeout: 10 * time.Millisecond, silenceTimeout: time.Second,
				maxHistory: defaultMaxHistory, commandCh: make(chan struct{}, 1), stopCh: make(chan struct{}),
				workerDone: make(chan struct{}), state: &endOfSpeechState{},
			}
			go endOfSpeech.worker()
			t.Cleanup(func() { require.NoError(t, endOfSpeech.Close(context.Background())) })
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{
				ContextID: "error-fallback", Script: "committed text",
			}))
			select {
			case packet := <-completed:
				assert.Equal(t, "committed text", packet.Speech)
			case <-time.After(500 * time.Millisecond):
				t.Fatal("prediction failure did not use the minimum delay")
			}
			require.NoError(t, endOfSpeech.Close(context.Background()))
			if testCase.loggedError == nil {
				assert.Empty(t, failures)
				return
			}
			require.Len(t, failures, 1)
			failure := <-failures
			assert.Equal(t, "turn prediction failed", failure.Record.Message)
			assert.Equal(t, "error-fallback", failure.ContextID)
			assert.Equal(t, "predict_end_of_turn", failure.Record.Attributes["operation"])
			assert.Contains(t, failure.Record.Attributes["error"], testCase.loggedError.Error())
		})
	}
}

func TestPredictionErrorCallbackCanReplaceTurn(t *testing.T) {
	completed := make(chan internal_type.EndOfSpeechPacket, 2)
	replaced := make(chan error, 1)
	endOfSpeech := &livekitEndOfSpeech{
		predictor: testPredictor{predict: func(string) (float64, error) {
			return 0, errTurnDetectorRunInference
		}},
		threshold: defaultThreshold, maxHistory: defaultMaxHistory,
		commandCh: make(chan struct{}, 1), stopCh: make(chan struct{}),
		workerDone: make(chan struct{}), state: &endOfSpeechState{},
	}
	endOfSpeech.onPacket = func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			switch packet := packet.(type) {
			case internal_type.ObservabilityLogRecordPacket:
				if packet.Record.Level == observability.LevelError {
					replaced <- endOfSpeech.Execute(ctx, internal_type.UserTextReceivedPacket{Text: "replacement"})
				}
			case internal_type.EndOfSpeechPacket:
				completed <- packet
			}
		}
		return nil
	}
	go endOfSpeech.worker()
	t.Cleanup(func() { require.NoError(t, endOfSpeech.Close(context.Background())) })
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: "obsolete"}))
	select {
	case err := <-replaced:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("error callback blocked turn replacement")
	}
	select {
	case packet := <-completed:
		assert.Equal(t, "replacement", packet.Speech)
	case <-time.After(time.Second):
		t.Fatal("replacement turn did not complete")
	}
	require.NoError(t, endOfSpeech.Close(context.Background()))
	assert.Empty(t, completed)
}
