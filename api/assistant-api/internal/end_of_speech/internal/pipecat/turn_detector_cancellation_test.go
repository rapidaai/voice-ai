//go:build integration && cgo

// Copyright (c) 2023-2025 RapidaAI
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_pipecat

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPipecatDetectorCancellation(t *testing.T) {
	// ONNX IR 8/opset 13: Loop carries 0.75 for max(0, max(input_features))*1e9 iterations.
	// Silence or zero features finish immediately; a positive feature exercises native cancellation.
	model, err := base64.StdEncoding.DecodeString(
		"CAg69QMKMgoOaW5wdXRfZmVhdHVyZXMSBHBlYWsiCVJlZHVjZU1heCoPCghrZWVwZGltcxgAoAECChYKBHBlYWsSCHBvc2l0aXZlIgRSZWx1Ch0KCHBvc2l0aXZlCgVzY2FsZRIFY291bnQiA011bAofCgVjb3VudBIFdHJpcHMiBENhc3QqCQoCdG8YB6ABAgrjAQoFdHJpcHMKCWNvbmRpdGlvbgoHaW5pdGlhbBIGbG9naXRzIgRMb29wKrcBCgRib2R5MqsBCh0KB2NvbmRfaW4SCGNvbmRfb3V0IghJZGVudGl0eQofCgh2YWx1ZV9pbhIJdmFsdWVfb3V0IghJZGVudGl0eRIEYm9keVoLCgFpEgYKBAgHEgBaEQoHY29uZF9pbhIGCgQICRIAWhYKCHZhbHVlX2luEgoKCAgBEgQKAggBYhIKCGNvbmRfb3V0EgYKBAgJEgBiFwoJdmFsdWVfb3V0EgoKCAgBEgQKAggBoAEFEgxjYW5jZWxsYXRpb24qDxABIgQoa25OQgVzY2FsZSoQEAkqAQFCCWNvbmRpdGlvbioTCAEQASIEAABAP0IHaW5pdGlhbFolCg5pbnB1dF9mZWF0dXJlcxITChEIARINCgIIAQoCCFAKAwigBmIUCgZsb2dpdHMSCgoICAESBAoCCAFCBAoAEA0=")
	require.NoError(t, err)
	modelPath := filepath.Join(t.TempDir(), "loop.onnx")
	require.NoError(t, os.WriteFile(modelPath, model, 0o600))
	pd, err := NewPipecatDetector(PipecatDetectorConfig{ModelPath: modelPath})
	require.NoError(t, err)
	t.Cleanup(pd.Destroy)
	audio := make([]float32, 16000)

	t.Run("before_start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		prob, err := pd.PredictContext(ctx, audio)
		require.Zero(t, prob)
		require.ErrorIs(t, err, context.Canceled)

		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		prob, err = pd.PredictContext(ctx, audio)
		require.Zero(t, prob)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		prob, err = pd.Predict(audio)
		require.NoError(t, err)
		require.Equal(t, 0.75, prob)
	})

	for _, deadline := range []bool{false, true} {
		name := "during_run_cancel"
		if deadline {
			name = "during_run_deadline"
		}
		t.Run(name, func(t *testing.T) {
			features := make([]float32, whisperNMels*whisperMaxFrames)
			features[0] = 1
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wantErr := context.Canceled
			if deadline {
				ctx, cancel = context.WithTimeout(ctx, 25*time.Millisecond)
				defer cancel()
				wantErr = context.DeadlineExceeded
			} else {
				timer := time.AfterFunc(25*time.Millisecond, cancel)
				defer timer.Stop()
			}
			started := time.Now()
			prob, err := pd.inferContext(ctx, features)
			require.Zero(t, prob)
			require.ErrorIs(t, err, wantErr)
			require.Less(t, time.Since(started), time.Second)

			features[0] = 0
			prob, err = pd.infer(features)
			require.NoError(t, err)
			require.Equal(t, 0.75, prob)
			prob, err = pd.PredictContext(context.Background(), audio)
			require.NoError(t, err)
			require.Equal(t, 0.75, prob)
		})
	}

	t.Run("prediction_error", func(t *testing.T) {
		prob, err := pd.PredictContext(context.Background(), nil)
		require.Zero(t, prob)
		require.ErrorIs(t, err, errPipecatDetectorEmptyAudio)

		prob, err = (*PipecatDetector)(nil).PredictContext(context.Background(), audio)
		require.Zero(t, prob)
		require.ErrorIs(t, err, errPipecatDetectorNil)

		prob, err = pd.infer(make([]float32, 1))
		require.Zero(t, prob)
		require.ErrorIs(t, err, errPipecatDetectorCreateInputTensor)

		prob, err = pd.Predict(audio)
		require.NoError(t, err)
		require.Equal(t, 0.75, prob)
	})
}
