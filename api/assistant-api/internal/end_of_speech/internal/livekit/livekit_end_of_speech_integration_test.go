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
			nativePredictor := nativeEndOfSpeech.predictor
			predictionCount := 0
			nativeEndOfSpeech.predictor = testPredictor{predictContext: func(ctx context.Context, text string) (float64, error) {
				probability, predictionError := nativePredictor.PredictContext(ctx, text)
				require.NoError(t, predictionError)
				require.False(t, math.IsNaN(probability))
				require.GreaterOrEqual(t, probability, 0.0)
				require.LessOrEqual(t, probability, 1.0)
				t.Logf("model=%s probability=%.6f prompt=%q", modelType, probability, text)
				predictionCount++
				return probability, predictionError
			}}
			t.Cleanup(func() {
				nativeEndOfSpeech.predictor = nativePredictor
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
				case packet := <-completed:
					require.Equal(t, speech, packet.Speech)
					require.Len(t, packet.Speechs, 1)
				case <-time.After(5 * time.Second):
					t.Fatal("native EOS model flow did not complete")
				}
			}
			require.Equal(t, 2, predictionCount)
			select {
			case packet := <-completed:
				t.Fatalf("duplicate native completion: %+v", packet)
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
}

func TestLivekitEndOfSpeech_EndpointingOptionPrecedence(t *testing.T) {
	for _, test := range []struct {
		name     string
		options  utils.Option
		expected time.Duration
	}{
		{name: "default", expected: 250 * time.Millisecond},
		{name: "legacy", options: utils.Option{optKeyLegacyTimeout: 70.0}, expected: 70 * time.Millisecond},
		{name: "fallback", options: utils.Option{optKeyLegacyTimeout: 70.0, optKeyFallbackTimeout: 80.0}, expected: 80 * time.Millisecond},
		{name: "quick", options: utils.Option{optKeyLegacyTimeout: 70.0, optKeyFallbackTimeout: 80.0, optKeyQuickTimeout: 90.0}, expected: 90 * time.Millisecond},
		{name: "stored strings", options: utils.Option{internal_options.MicrophoneEOSOptionQuickTimeout: "900"}, expected: 900 * time.Millisecond},
		{name: "invalid quick uses fallback", options: utils.Option{internal_options.MicrophoneEOSOptionQuickTimeout: "invalid", internal_options.MicrophoneEOSOptionFallbackTimeout: "800"}, expected: 800 * time.Millisecond},
		{name: "hex integer uses fallback", options: utils.Option{optKeyQuickTimeout: "0x10", optKeyFallbackTimeout: "800"}, expected: 800 * time.Millisecond},
		{name: "padded quick uses fallback", options: utils.Option{optKeyQuickTimeout: " 900 ", optKeyFallbackTimeout: "800"}, expected: 800 * time.Millisecond},
		{name: "invalid first alias uses timeout", options: utils.Option{optKeyFallbackTimeout: "0x10", optKeyLegacyTimeout: "700"}, expected: 700 * time.Millisecond},
		{name: "exponent", options: utils.Option{optKeyQuickTimeout: ".8e3", optKeyFallbackTimeout: "900"}, expected: 800 * time.Millisecond},
		{name: "digit separators", options: utils.Option{optKeyQuickTimeout: "1_000", optKeyFallbackTimeout: "900"}, expected: time.Second},
		{name: "hex float", options: utils.Option{optKeyQuickTimeout: "0x1p8", optKeyFallbackTimeout: "900"}, expected: 256 * time.Millisecond},
	} {
		t.Run(test.name, func(t *testing.T) {
			endOfSpeech, err := New(
				WithContext(t.Context()),
				WithOptions(test.options),
				WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }),
			)
			require.NoError(t, err)
			defer endOfSpeech.Close(context.Background())
			require.Equal(t, test.expected, endOfSpeech.(*livekitEndOfSpeech).quickTimeout)
		})
	}
}
