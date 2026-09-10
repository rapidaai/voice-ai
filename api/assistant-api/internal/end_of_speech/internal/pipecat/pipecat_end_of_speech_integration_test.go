//go:build integration

package internal_pipecat

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEOS_UIOptionDefaults(t *testing.T) {
	content, err := os.ReadFile("../../../../../../ui/src/providers/pipecat_smart_turn_eos/eos.json")
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
	require.NotContains(t, settings, internal_options.MicrophoneEOSOptionQuickTimeout)
	endOfSpeech, err := New(
		WithContext(t.Context()),
		WithOptions(settings),
		WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }),
	)
	require.NoError(t, err)
	defer endOfSpeech.Close(context.Background())
	configured := endOfSpeech.(*pipecatEndOfSpeech)
	require.Equal(t, defaultPctThreshold, configured.threshold)
	require.Equal(t, time.Duration(defaultPctFallbackTimeout)*time.Millisecond, configured.fallbackTimeout)
	require.Equal(t, time.Duration(defaultPctExtendedTimeout)*time.Millisecond, configured.extendedTimeout)
}

func TestEOS_CanonicalConstructorOptions(t *testing.T) {
	for _, test := range []struct {
		name            string
		settings        utils.Option
		threshold       float64
		fallbackTimeout time.Duration
		extendedTimeout time.Duration
	}{
		{
			name:            "missing canonical values use defaults",
			threshold:       defaultPctThreshold,
			fallbackTimeout: 500 * time.Millisecond,
			extendedTimeout: 3000 * time.Millisecond,
		},
		{
			name: "custom canonical values",
			settings: utils.Option{
				internal_options.MicrophoneEOSOptionThreshold:       "0.75",
				internal_options.MicrophoneEOSOptionFallbackTimeout: "8e2",
				internal_options.MicrophoneEOSOptionExtendedTimeout: "1_800",
			},
			threshold:       0.75,
			fallbackTimeout: 800 * time.Millisecond,
			extendedTimeout: 1800 * time.Millisecond,
		},
		{
			name: "zero canonical values",
			settings: utils.Option{
				internal_options.MicrophoneEOSOptionThreshold:       0.0,
				internal_options.MicrophoneEOSOptionFallbackTimeout: 0.0,
				internal_options.MicrophoneEOSOptionExtendedTimeout: 0.0,
			},
			threshold:       0,
			fallbackTimeout: 0,
			extendedTimeout: 0,
		},
		{
			name: "invalid canonical values use defaults",
			settings: utils.Option{
				internal_options.MicrophoneEOSOptionThreshold:       "invalid",
				internal_options.MicrophoneEOSOptionFallbackTimeout: "invalid",
				internal_options.MicrophoneEOSOptionExtendedTimeout: "invalid",
			},
			threshold:       defaultPctThreshold,
			fallbackTimeout: 500 * time.Millisecond,
			extendedTimeout: 3000 * time.Millisecond,
		},
		{
			name: "aliases ignored",
			settings: utils.Option{
				internal_options.MicrophoneEOSOptionTimeout:      "700",
				"microphone.eos.silence_timeout":                 "1800",
				internal_options.MicrophoneEOSOptionQuickTimeout: "900",
				internal_options.MicrophoneEOSOptionModel:        "ignored",
			},
			threshold:       defaultPctThreshold,
			fallbackTimeout: 500 * time.Millisecond,
			extendedTimeout: 3000 * time.Millisecond,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			endOfSpeech, err := New(
				WithContext(t.Context()),
				WithOptions(test.settings),
				WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }),
			)
			require.NoError(t, err)
			defer endOfSpeech.Close(context.Background())
			configured := endOfSpeech.(*pipecatEndOfSpeech)
			require.Equal(t, test.threshold, configured.threshold)
			require.Equal(t, test.fallbackTimeout, configured.fallbackTimeout)
			require.Equal(t, test.extendedTimeout, configured.extendedTimeout)
		})
	}
}

func TestEOS_NativeSmartTurnAudioFlow(t *testing.T) {
	completed := make(chan internal_type.EndOfSpeechPacket, 4)
	endOfSpeech, err := New(
		WithContext(t.Context()),
		WithOptions(utils.Option{
			"microphone.eos.fallback_timeout": 50.0,
			"microphone.eos.extended_timeout": 100.0,
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
	nativeEndOfSpeech := endOfSpeech.(*pipecatEndOfSpeech)
	nativePredictor := nativeEndOfSpeech.predictor
	var predictionCount int
	nativeEndOfSpeech.predictor = testPredictor{predict: func(audio []float32) (float64, error) {
		probability, predictionError := nativePredictor.Predict(audio)
		require.NoError(t, predictionError)
		predictionCount++
		return probability, predictionError
	}}
	t.Cleanup(func() {
		nativeEndOfSpeech.predictor = nativePredictor
		require.NoError(t, endOfSpeech.Close(context.Background()))
	})
	for _, speech := range []string{"first native turn", "second native turn"} {
		require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
			Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		}))
		require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(16000)))
		require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput(speech, true)))
		select {
		case packet := <-completed:
			t.Fatalf("turn completed while VAD was speaking: %+v", packet)
		default:
		}
		require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
			Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
		}))
		require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.EndOfSpeechAudioPacket{
			Audio: make([]byte, 3200),
		}))
		select {
		case packet := <-completed:
			assert.Equal(t, speech, packet.Speech)
			require.Len(t, packet.Speechs, 1)
			assert.False(t, packet.Speechs[0].Interim)
		case <-time.After(5 * time.Second):
			t.Fatal("native model flow did not complete")
		}
	}
	assert.Equal(t, 2, predictionCount)
	select {
	case packet := <-completed:
		t.Fatalf("duplicate native completion: %+v", packet)
	case <-time.After(100 * time.Millisecond):
	}
}
