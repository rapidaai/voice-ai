// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package nvidia_internal

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNvidiaProcessor(t *testing.T) {
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	processor := NewNvidiaProcessor(logger, utils.Option{})
	require.NotNil(t, processor)
	_, ok := processor.(*nvidiaProcessor)
	assert.True(t, ok)
}

func TestProcess_Passthrough(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewNvidiaProcessor(logger, utils.Option{})

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"simple text", "Hello world"},
		{"markdown preserved", "# Header **bold**"},
		{"xml chars preserved", "Tom & Jerry a < b > c"},
		{"whitespace preserved", "Hello    world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.input, result)
		})
	}
}

func TestProcess_NoSSML(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewNvidiaProcessor(logger, utils.Option{})

	result := processor.Process("Tom & Jerry")
	assert.NotContains(t, result, "&amp;")
	assert.NotContains(t, result, "<break")
}

func BenchmarkProcess(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewNvidiaProcessor(logger, utils.Option{})
	text := "Hello, this is a simple text for TTS processing."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(text)
	}
}
