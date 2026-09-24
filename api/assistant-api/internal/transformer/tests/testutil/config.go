// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package transformer_testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"
)

// TransformerTestConfig is the top-level configuration for integration tests.
type TransformerTestConfig struct {
	TTS map[string]ProviderConfig `yaml:"tts"`
	STT map[string]ProviderConfig `yaml:"stt"`
}

// ProviderConfig holds credentials and options for a single provider.
type ProviderConfig struct {
	Enabled    bool                   `yaml:"enabled"`
	Credential map[string]string      `yaml:"credential"`
	Options    map[string]interface{} `yaml:"options"`
}

// testdataDir returns the absolute path to the testdata directory relative to this file.
func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "testdata")
}

// LoadConfig reads integration_config.yaml from testdata/ and parses it.
// The config file path can be overridden via TRANSFORMER_TEST_CONFIG env var.
func LoadConfig(t *testing.T) *TransformerTestConfig {
	t.Helper()
	configPath := os.Getenv("TRANSFORMER_TEST_CONFIG")
	if configPath == "" {
		configPath = filepath.Join(testdataDir(), "integration_config.yaml")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Skipf("integration config not found at %s: %v (create from integration_config.yaml.example)", configPath, err)
	}
	var cfg TransformerTestConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse integration config: %v", err)
	}
	return &cfg
}

// TTSProvider returns the ProviderConfig for the given TTS provider name.
// It skips the test if the provider is not configured or not enabled.
func (c *TransformerTestConfig) TTSProvider(t *testing.T, name string) ProviderConfig {
	t.Helper()
	p, ok := c.TTS[name]
	if !ok || !p.Enabled {
		t.Skipf("TTS provider %q not configured or disabled", name)
	}
	return p
}

// STTProvider returns the ProviderConfig for the given STT provider name.
// It skips the test if the provider is not configured or not enabled.
func (c *TransformerTestConfig) STTProvider(t *testing.T, name string) ProviderConfig {
	t.Helper()
	p, ok := c.STT[name]
	if !ok || !p.Enabled {
		t.Skipf("STT provider %q not configured or disabled", name)
	}
	return p
}
