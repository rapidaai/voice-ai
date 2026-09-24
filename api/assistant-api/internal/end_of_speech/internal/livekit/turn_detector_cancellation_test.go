// Copyright (c) 2023-2025 RapidaAI
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_livekit

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTurnDetectorCancellation(t *testing.T) {
	// ONNX IR 8/opset 13: Loop runs sum(input_ids) times, carrying a [1,n] tensor of 0.75.
	// Special token IDs select one iteration for reuse or a billion for cancellation.
	model, err := base64.StdEncoding.DecodeString(
		"CAg60QMKLgoJaW5wdXRfaWRzEgV0cmlwcyIJUmVkdWNlU3VtKg8KCGtlZXBkaW1zGACgAQIKGQoJaW5wdXRfaWRzEgVzaGFwZSIFU2hhcGUKPwoFc2hhcGUSB2luaXRpYWwiD0NvbnN0YW50T2ZTaGFwZSocCgV2YWx1ZSoQCAEQASIEAABAP0IEZmlsbKABBArrAQoFdHJpcHMKCWNvbmRpdGlvbgoHaW5pdGlhbBIEcHJvYiIETG9vcCrBAQoEYm9keTK1AQodCgdjb25kX2luEghjb25kX291dCIISWRlbnRpdHkKHwoIdmFsdWVfaW4SCXZhbHVlX291dCIISWRlbnRpdHkSBGJvZHlaCwoBaRIGCgQIBxIAWhEKB2NvbmRfaW4SBgoECAkSAFobCgh2YWx1ZV9pbhIPCg0IARIJCgIIAQoDEgFuYhIKCGNvbmRfb3V0EgYKBAgJEgBiHAoJdmFsdWVfb3V0Eg8KDQgBEgkKAggBCgMSAW6gAQUSDGNhbmNlbGxhdGlvbioQEAkqAQFCCWNvbmRpdGlvblocCglpbnB1dF9pZHMSDwoNCAcSCQoCCAEKAxIBbmIXCgRwcm9iEg8KDQgBEgkKAggBCgMSAW5CBAoAEA0=")
	require.NoError(t, err)
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "loop.onnx")
	tokenizerPath := filepath.Join(dir, "tokenizer.json")
	require.NoError(t, os.WriteFile(modelPath, model, 0o600))
	require.NoError(t, os.WriteFile(tokenizerPath, []byte(`{
		"model":{"vocab":{},"merges":[]},
		"added_tokens":[
			{"id":1,"content":"<fast>","special":true},
			{"id":1000000000,"content":"<slow>","special":true}
		]
	}`), 0o600))

	for _, modelType := range []string{"en", "multilingual"} {
		t.Run(modelType, func(t *testing.T) {
			td, err := NewTurnDetector(TurnDetectorConfig{
				ModelPath: modelPath, TokenizerPath: tokenizerPath, ModelType: modelType,
			})
			if errors.Is(err, errTurnDetectorRuntimeAPIUnavailable) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				prob, predictErr := td.PredictContext(ctx, "<fast>")
				require.Zero(t, prob)
				require.ErrorIs(t, predictErr, errTurnDetectorRuntimeAPIUnavailable)
				prob, predictErr = td.Predict("<fast>")
				require.Zero(t, prob)
				require.ErrorIs(t, predictErr, errTurnDetectorRuntimeAPIUnavailable)
				t.Skip("native runtime unavailable; nocgo prediction contract verified")
			}
			require.NoError(t, err)
			defer td.Destroy()

			t.Run("before_start", func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				prob, err := td.PredictContext(ctx, "<slow>")
				require.Zero(t, prob)
				require.ErrorIs(t, err, context.Canceled)

				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				defer cancel()
				prob, err = td.PredictContext(ctx, "<slow>")
				require.Zero(t, prob)
				require.ErrorIs(t, err, context.DeadlineExceeded)

				prob, err = td.Predict("<fast>")
				require.NoError(t, err)
				require.Equal(t, 0.75, prob)
			})

			for _, deadline := range []bool{false, true} {
				name := "during_run_cancel"
				if deadline {
					name = "during_run_deadline"
				}
				t.Run(name, func(t *testing.T) {
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
					prob, err := td.PredictContext(ctx, "<slow>")
					require.Zero(t, prob)
					require.ErrorIs(t, err, wantErr)
					require.Less(t, time.Since(started), time.Second)

					prob, err = td.PredictContext(context.Background(), "<fast>")
					require.NoError(t, err)
					require.Equal(t, 0.75, prob)
				})
			}

			t.Run("prediction_error", func(t *testing.T) {
				prob, err := td.PredictContext(context.Background(), "")
				require.Zero(t, prob)
				require.ErrorIs(t, err, errTurnDetectorEmptyTokenSequence)
			})
		})
	}
}
