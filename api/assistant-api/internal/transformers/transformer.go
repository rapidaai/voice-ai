// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformers

import (
	"context"

	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/utils"
)

// Transformers is a generic interface that defines a transform method
// for converting one type to another.
//
// Type parameters:
//   - IN: The input type for the transformation.
//   - OUT: The output type for the transformation.
//
// The transform method takes an input of type IN and returns an output
// of type OUT along with an error. This allows for flexible type
// conversion and data transformation while providing error handling.
//
// Implementations of this interface can be used to create reusable
// and composable transformation logic for various data types and
// structures within an application.

type Transformers[IN any, opts TransformOption] interface {
	// The `Initialize() error` method in the `Transformers` interface is defining a function signature
	// for a method that initializes or sets up any necessary resources or configurations before the
	// transformation process begins. This method is expected to return an error if any issues occur
	// during the initialization process, allowing for proper error handling and ensuring that the
	// transformation can proceed only when the initialization is successful.
	Initialize() error

	// The comment `// Transformers[[]byte, string]` is specifying the type parameters for the interface
	// `SpeechToTextTransformer`. It is indicating that `SpeechToTextTransformer` extends the `Transformers`
	// interface with the specific type parameters `[]byte` as the input type and `string` as the output
	// type for the transformation.
	Transform(context.Context, IN, opts) error

	//
	// The `Cancel() error` method in the `Transformers` interface defines a function signature for a
	// method that aborts or cancels an ongoing transformation process. This method is expected to
	// return an error if any issues occur during the cancellation process.
	Close(context.Context) error
}

// SpeechToTextTransformer is an interface for transforming input audio data.
// It extends the Transformers interface, specifying that it transforms
// from []byte (raw audio data) to string (processed audio representation).
type SpeechToTextTransformer interface {
	// The `Name() string` method is defining a function signature for a method that returns a string
	// representing the name of the transformer. This method is expected to provide a human-readable
	// identifier for the transformer implementation, allowing users to easily identify and differentiate
	// between different transformer instances based on their names.
	Name() string

	//
	Transformers[[]byte, *SpeechToTextOption]
}

// OutputAudioTransformer is an interface for transforming output audio data.
// It extends the Transformers interface, specifying that it transforms
// from string (processed audio representation) to []byte (raw audio data).
type TextToSpeechTransformer interface {
	Name() string

	//
	Transformers[string, *TextToSpeechOption]
}

// SpeechToTextTransformerOptions defines the interface for handling audio transformation events.
// It provides callbacks for when transcripts are generated and when the transformation process is complete.
//
// The interface includes two methods:
//   - OnTranscript: Called when a new transcript is available. It receives the transcript text
//     and a boolean indicating whether the transcript is complete.
//   - OnComplete: Called when the entire audio transformation process is finished.
//
// Implementations of this interface can be used to handle real-time updates during
// audio processing, allowing for actions such as displaying interim results,
// updating progress indicators, or triggering subsequent processing steps.
type SpeechToTextInitializeOptions struct {

	// audio config
	AudioConfig *internal_voices.AudioConfig

	//
	// on transcript
	OnTranscript func(
		transcript string,
		confidence float64,
		languages string,
		isCompleted bool,
	) error

	// options of model
	ModelOptions utils.Option
}

// OutputAudioTransformerOptions defines the interface for handling audio output transformation
type TextToSpeechInitializeOptions struct {

	// audio config
	AudioConfig *internal_voices.AudioConfig

	// OnSpeech is called when speech is detected in the audio stream
	// It receives a byte slice containing the speech audio data
	// Returns an error if there's an issue processing the speech
	OnSpeech func(string, []byte) error

	// OnComplete is called when the audio transformation is complete
	// Returns an error if there's an issue finalizing the transformation
	OnComplete func(string) error

	// options of model
	ModelOptions utils.Option
}

type TransformOption interface{}

type TextToSpeechOption struct {
	TransformOption
	ContextId  string
	IsComplete bool
}

type SpeechToTextOption struct {
	TransformOption
}
