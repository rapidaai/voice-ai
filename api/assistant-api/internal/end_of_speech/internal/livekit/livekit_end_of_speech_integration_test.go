//go:build integration

package internal_livekit

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
)

func TestLivekitEndOfSpeech_UIOptionDefaults(t *testing.T) {
	content, err := os.ReadFile("../../../../../../ui/src/providers/livekit_eos/eos.json")
	require.NoError(t, err)
	var config struct {
		Parameters []struct {
			Key     string `json:"key"`
			Default any    `json:"default"`
		} `json:"parameters"`
	}
	require.NoError(t, json.Unmarshal(content, &config))
	settings := utils.Option{}
	for _, parameter := range config.Parameters {
		settings[parameter.Key] = parameter.Default
	}
	require.NotContains(t, settings, internal_options.MicrophoneEOSOptionFallbackTimeout)
	endOfSpeech, err := New(
		WithContext(t.Context()),
		WithOptions(settings),
		WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }),
	)
	require.NoError(t, err)
	defer endOfSpeech.Close(context.Background())
	configured := endOfSpeech.(*livekitEndOfSpeech)
	require.Equal(t, defaultThreshold, configured.threshold)
	require.Equal(t, time.Duration(defaultQuickTimeout)*time.Millisecond, configured.quickTimeout)
	require.Equal(t, time.Duration(defaultSilenceTimeout)*time.Millisecond, configured.silenceTimeout)
	require.Equal(t, int(defaultMaxHistory), configured.maxHistory)
	require.Equal(t, defaultModelType, configured.modelType)
}

func TestLivekitEndOfSpeech_NativeModelFlow(t *testing.T) {
	for _, modelType := range []string{defaultModelType, multilingualModelType} {
		t.Run(modelType, func(t *testing.T) {
			if _, err := os.Stat(resolveModelPath("", modelType == multilingualModelType)); err != nil {
				t.Skipf("livekit %s model asset unavailable: %v", modelType, err)
			}
			if _, err := os.Stat(resolveTokenizerPath("")); err != nil {
				t.Skipf("livekit tokenizer asset unavailable: %v", err)
			}

			completed := make(chan internal_type.EndOfSpeechPacket, 4)
			endOfSpeech, err := New(
				WithContext(t.Context()),
				WithOptions(utils.Option{
					optKeyModel:           modelType,
					optKeyQuickTimeout:    50.0,
					optKeyExtendedTimeout: 100.0,
				}),
				WithOnPacket(func(ctx context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
							completed <- result
						}
					}
					return nil
				}),
			)
			require.NoError(t, err)
			nativeEndOfSpeech := endOfSpeech.(*livekitEndOfSpeech)
			nativePredictor := nativeEndOfSpeech.predictor.(*TurnDetector)
			type predictionResult struct {
				probability float64
				err         error
			}
			predictionResults := make(chan predictionResult, 2)
			nativeEndOfSpeech.predictor = testPredictor{
				predictContext: func(ctx context.Context, text string) (float64, error) {
					probability, predictionError := nativePredictor.PredictContext(ctx, text)
					predictionResults <- predictionResult{probability: probability, err: predictionError}
					return probability, predictionError
				},
				destroy: nativePredictor.Destroy,
			}
			t.Cleanup(func() {
				require.NoError(t, endOfSpeech.Close(context.Background()))
			})
			for _, speech := range []string{"I just wanted to talk to you and then", "Thank you. Goodbye."} {
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				}))
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.SpeechToTextPacket{Script: speech}))
				require.Empty(t, completed)
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
				}))
				select {
				case result := <-predictionResults:
					require.NoError(t, result.err)
					require.False(t, math.IsNaN(result.probability))
					require.GreaterOrEqual(t, result.probability, 0.0)
					require.LessOrEqual(t, result.probability, 1.0)
				case <-time.After(5 * time.Second):
					t.Fatal("native prediction did not finish")
				}
				select {
				case packet := <-completed:
					require.Equal(t, speech, packet.Speech)
					require.Len(t, packet.Speechs, 1)
				case <-time.After(5 * time.Second):
					t.Fatal("native EOS model flow did not complete")
				}
			}
			require.NoError(t, endOfSpeech.Close(context.Background()))
			require.Empty(t, predictionResults, "unexpected extra native predictions")
			select {
			case packet := <-completed:
				t.Fatalf("duplicate native completion: %+v", packet)
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
}

func TestLivekitEndOfSpeech_CanonicalConstructorOptions(t *testing.T) {
	for _, test := range []struct {
		name            string
		options         utils.Option
		threshold       float64
		quickTimeout    time.Duration
		extendedTimeout time.Duration
		maxHistory      int
		modelType       string
		expectedError   error
	}{
		{
			name:            "defaults",
			threshold:       defaultThreshold,
			quickTimeout:    time.Duration(defaultQuickTimeout) * time.Millisecond,
			extendedTimeout: time.Duration(defaultSilenceTimeout) * time.Millisecond,
			maxHistory:      int(defaultMaxHistory),
			modelType:       defaultModelType,
		},
		{
			name: "custom canonical values",
			options: utils.Option{
				optKeyThreshold:       "0.75",
				optKeyQuickTimeout:    "9e2",
				optKeyExtendedTimeout: "1_200",
				optKeyMaxHistory:      "3",
				optKeyModel:           "manual-livekit",
			},
			threshold:       0.75,
			quickTimeout:    900 * time.Millisecond,
			extendedTimeout: 1200 * time.Millisecond,
			maxHistory:      3,
			modelType:       "manual-livekit",
		},
		{
			name: "zero canonical values",
			options: utils.Option{
				optKeyThreshold:       0.0,
				optKeyQuickTimeout:    0.0,
				optKeyExtendedTimeout: 0.0,
			},
			threshold:       0,
			quickTimeout:    0,
			extendedTimeout: 0,
			maxHistory:      int(defaultMaxHistory),
			modelType:       defaultModelType,
		},
		{
			name: "invalid canonical values return an error",
			options: utils.Option{
				optKeyThreshold:       "invalid",
				optKeyQuickTimeout:    "invalid",
				optKeyExtendedTimeout: "invalid",
				optKeyMaxHistory:      "invalid",
				optKeyModel:           "",
			},
			expectedError: errLivekitInvalidOption,
		},
		{
			name: "aliases ignored",
			options: utils.Option{
				internal_options.MicrophoneEOSOptionFallbackTimeout: 700.0,
				internal_options.MicrophoneEOSOptionTimeout:         800.0,
				"microphone.eos.silence_timeout":                    900.0,
			},
			threshold:       defaultThreshold,
			quickTimeout:    time.Duration(defaultQuickTimeout) * time.Millisecond,
			extendedTimeout: time.Duration(defaultSilenceTimeout) * time.Millisecond,
			maxHistory:      int(defaultMaxHistory),
			modelType:       defaultModelType,
		},
		{
			name: "aliases do not rescue invalid canonical values",
			options: utils.Option{
				optKeyQuickTimeout:    "invalid",
				optKeyExtendedTimeout: "invalid",
				internal_options.MicrophoneEOSOptionFallbackTimeout: "700",
				internal_options.MicrophoneEOSOptionTimeout:         "800",
				"microphone.eos.silence_timeout":                    "900",
			},
			expectedError: errLivekitInvalidOption,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			endOfSpeech, err := New(
				WithContext(t.Context()),
				WithOptions(test.options),
				WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }),
			)
			if test.expectedError != nil {
				require.ErrorIs(t, err, test.expectedError)
				require.Nil(t, endOfSpeech)
				return
			}
			require.NoError(t, err)
			defer endOfSpeech.Close(context.Background())
			configured := endOfSpeech.(*livekitEndOfSpeech)
			require.Equal(t, test.threshold, configured.threshold)
			require.Equal(t, test.quickTimeout, configured.quickTimeout)
			require.Equal(t, test.extendedTimeout, configured.silenceTimeout)
			require.Equal(t, test.maxHistory, configured.maxHistory)
			require.Equal(t, test.modelType, configured.modelType)
		})
	}
}

func TestLivekitEndOfSpeech_InvalidConfiguredPaths(t *testing.T) {
	for _, test := range []struct {
		name    string
		options utils.Option
	}{
		{name: "model path", options: utils.Option{optKeyModelPath: "/nonexistent/model.onnx"}},
		{name: "tokenizer path", options: utils.Option{optKeyTokenizerPath: "/nonexistent/tokenizer.json"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			endOfSpeech, err := New(
				WithContext(t.Context()),
				WithOptions(test.options),
				WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }),
			)
			require.Error(t, err)
			require.Nil(t, endOfSpeech)
		})
	}
}
