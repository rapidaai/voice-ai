package options

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEOSProviderOptionRegistry(t *testing.T) {
	for _, provider := range []struct {
		name string
		keys map[string]bool
	}{
		{
			name: "pipecat_smart_turn_eos",
			keys: map[string]bool{
				MicrophoneEOSOptionFallbackTimeout: true,
				MicrophoneEOSOptionThreshold:       true,
				MicrophoneEOSOptionExtendedTimeout: true,
			},
		},
		{
			name: "livekit_eos",
			keys: map[string]bool{
				MicrophoneEOSOptionThreshold:       true,
				MicrophoneEOSOptionQuickTimeout:    true,
				MicrophoneEOSOptionExtendedTimeout: true,
				MicrophoneEOSOptionModel:           true,
				MicrophoneEOSOptionMaxHistoryTurns: true,
			},
		},
	} {
		t.Run(provider.name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("../../../../ui/src/providers", provider.name, "eos.json"))
			if err != nil {
				t.Fatal(err)
			}
			var config struct {
				Parameters []struct {
					Key string `json:"key"`
				} `json:"parameters"`
			}
			if err := json.Unmarshal(content, &config); err != nil {
				t.Fatal(err)
			}
			for _, parameter := range config.Parameters {
				if parameter.Key == MicrophoneEOSOptionTimeout || parameter.Key == "microphone.eos.silence_timeout" {
					t.Errorf("provider-specific EOS control uses non-provider timeout key: %s", parameter.Key)
				}
				if !provider.keys[parameter.Key] {
					t.Errorf("unregistered, unsupported, or duplicate EOS control: %s", parameter.Key)
				}
				delete(provider.keys, parameter.Key)
			}
			for key := range provider.keys {
				t.Errorf("missing supported EOS control: %s", key)
			}
		})
	}
}
