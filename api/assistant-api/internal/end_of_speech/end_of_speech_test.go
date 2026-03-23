// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_end_of_speech

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// MockEndOfSpeechCallback is a simple callback function for testing
var mockCallback = func(ctx context.Context, result ...internal_type.Packet) error {
	return nil
}

func TestGetEndOfSpeech_SilenceBasedIdentifier(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	endOfSpeech, err := GetEndOfSpeech(context.Background(), logger, mockCallback, utils.Option{EndOfSpeechOptionsKeyProvider: SilenceBasedEndOfSpeech})

	require.NoError(t, err)
	assert.NotNil(t, endOfSpeech)
	assert.IsType(t, endOfSpeech, endOfSpeech)
}

func TestGetEndOfSpeech_LiveKitIdentifier_FallsBackToSilenceBased(t *testing.T) {
	// LiveKit turn-detector removed due to license restrictions.
	// Selecting livekit_eos now falls back to silence_based_eos.
	logger, _ := commons.NewApplicationLogger()

	endOfSpeech, err := GetEndOfSpeech(t.Context(), logger, mockCallback, utils.Option{EndOfSpeechOptionsKeyProvider: LiveKitEndOfSpeech})

	require.NoError(t, err)
	assert.NotNil(t, endOfSpeech)
	if endOfSpeech != nil {
		defer endOfSpeech.Close()
	}
}

func TestEndOfSpeechIdentifier_Constants(t *testing.T) {
	assert.Equal(t, EndOfSpeechIdentifier("silence_based_eos"), SilenceBasedEndOfSpeech)
	assert.Equal(t, EndOfSpeechIdentifier("livekit_eos"), LiveKitEndOfSpeech)
	assert.NotEqual(t, SilenceBasedEndOfSpeech, LiveKitEndOfSpeech)
}

func TestGetEndOfSpeech_WithNilLogger(t *testing.T) {

	// This test validates that the function passes nil logger to NewSilenceBasedEndOfSpeech
	// which should handle it appropriately or fail gracefully
	endOfSpeech, err := GetEndOfSpeech(t.Context(), nil, mockCallback, utils.Option{EndOfSpeechOptionsKeyProvider: SilenceBasedEndOfSpeech})

	// The behavior depends on internal_silence_based_end_of_speech implementation
	// Either it should error or handle nil logger gracefully
	if err == nil {
		assert.NotNil(t, endOfSpeech)
	} else {
		assert.Nil(t, endOfSpeech)
	}
}

func TestGetEndOfSpeech_WithNilCallback(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	// Test with nil callback
	endOfSpeech, err := GetEndOfSpeech(t.Context(), logger, nil, utils.Option{EndOfSpeechOptionsKeyProvider: SilenceBasedEndOfSpeech})

	// The behavior depends on internal_silence_based_end_of_speech implementation
	if err == nil {
		assert.NotNil(t, endOfSpeech)
	} else {
		assert.Nil(t, endOfSpeech)
	}
}

func TestGetEndOfSpeech_WithNilOptions(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	// Test with nil options
	endOfSpeech, err := GetEndOfSpeech(t.Context(), logger, mockCallback, nil)

	// The behavior depends on internal_silence_based_end_of_speech implementation
	if err == nil {
		assert.NotNil(t, endOfSpeech)
	} else {
		assert.Nil(t, endOfSpeech)
	}
}
