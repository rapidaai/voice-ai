// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_end_of_speech

import (
	"context"

	internal_pipecat "github.com/rapidaai/api/assistant-api/internal/end_of_speech/internal/pipecat"
	internal_silence_based "github.com/rapidaai/api/assistant-api/internal/end_of_speech/internal/silence_based"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type EndOfSpeechIdentifier string

const (
	SilenceBasedEndOfSpeech       EndOfSpeechIdentifier = "silence_based_eos"
	PipecatSmartTurnEndOfSpeech   EndOfSpeechIdentifier = "pipecat_smart_turn_eos"
	EndOfSpeechOptionsKeyProvider                       = "microphone.eos.provider"

	// Removed LiveKit turn-detector
	// This was removed due to license restrictions requiring usage only within LiveKit Agents.
	// The implementation is retained in internal/livekit/ for benchmark and testing purposes only.
	LiveKitEndOfSpeech EndOfSpeechIdentifier = "livekit_eos"
)

func GetEndOfSpeech(ctx context.Context, logger commons.Logger, onCallback func(context.Context, ...internal_type.Packet) error, opts utils.Option) (internal_type.EndOfSpeech, error) {
	provider, _ := opts.GetString(EndOfSpeechOptionsKeyProvider)
	switch EndOfSpeechIdentifier(provider) {
	case SilenceBasedEndOfSpeech:
		return internal_silence_based.NewSilenceBasedEndOfSpeech(logger, onCallback, opts)
	case PipecatSmartTurnEndOfSpeech:
		return internal_pipecat.NewPipecatEndOfSpeech(logger, onCallback, opts)
	case LiveKitEndOfSpeech:
		// Removed: LiveKit turn-detector license restricts usage to LiveKit Agents only.
		// Falling back to silence-based EOS.
		logger.Warnf("livekit_eos is no longer available due to license restrictions, falling back to silence_based_eos")
		return internal_pipecat.NewPipecatEndOfSpeech(logger, onCallback, opts)
	default:
		return internal_silence_based.NewSilenceBasedEndOfSpeech(logger, onCallback, opts)
	}
}
