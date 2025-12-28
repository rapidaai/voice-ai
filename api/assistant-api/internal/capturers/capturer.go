// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_capturers

import (
	"context"

	type_enums "github.com/rapidaai/pkg/types/enums"
)

type CapturerOutput struct {
	Name     string                 `json:"name"`
	Paths    []string               `json:"paths"`
	Metadata map[string]interface{} `json:"metadata"`
}

type Capturer[t any] interface {
	Name() string
	Capture(ctx context.Context, role type_enums.MessageActor, s t) error
	Persist(ctx context.Context, key string) (*CapturerOutput, error)
}

/*
AudioCapture is an interface for analyzers that work with audio data.
It extends the base Capture interface.

Methods:
  - Name() string: Inherited from Capture, returns the name of the analyzer.
  - Capture(s []byte): Performs analysis on the given audio data represented as a byte slice.
*/
type AudioCapturer interface {
	Capturer[[]byte]
}

type TextCapturer interface {
	Capturer[string]
}
