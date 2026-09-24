// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_silence_based

import "time"

const (
	// Provider constants configure silence-based EOS at construction time.
	silenceBasedEndOfSpeechName = "silenceBasedEndOfSpeech"
	optSilenceTimeout           = "microphone.eos.timeout"
	defaultSilenceTimeout       = 1000 * time.Millisecond
	defaultVadEndTimeout        = 250 * time.Millisecond
)

const (
	vadStateIdle vadState = iota
	vadStateSpeaking
	vadStateEnded
)

const (
	transcriptStateIdle transcriptState = iota
	transcriptStateInterimPending
	transcriptStateFinalized
	transcriptStateFinalizedWithPendingInterim
)
