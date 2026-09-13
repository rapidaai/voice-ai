// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package lifecycle

import "errors"

var (
	ErrEmptyContextID              = errors.New("empty context id")
	ErrStaleContext                = errors.New("stale context")
	ErrInvalidTransition           = errors.New("invalid message lifecycle transition")
	ErrInvalidPlaybackControl      = errors.New("invalid playback control kind")
	ErrPlaybackTerminalNotIssued   = errors.New("playback terminal not issued")
	ErrDuplicatePlaybackCompletion = errors.New("duplicate playback completion")
	ErrSenderNotConfigured         = errors.New("message lifecycle output sender is not configured")
	ErrDispatcherNotConfigured     = errors.New("message lifecycle packet dispatcher is not configured")
)
