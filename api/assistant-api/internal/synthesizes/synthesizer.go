package internal_synthesizers

import (
	"context"

	"github.com/rapidaai/pkg/utils"
)

// Synthesizer is a generic interface that defines the contract for any type of synthesizer.
// It uses a generic type parameter IN to allow for different input types.
// The interface defines two methods: Synthesize and Flush.
type Synthesizer[IN any] interface {
	// Synthesize takes a context, a text string, and optional configuration.
	Synthesize(ctx context.Context, contextId string, text IN) IN
}

type SentenceSynthesizer interface {
	Synthesizer[string]
}

type SynthesizerOptions struct {
	SpeakerOptions utils.Option
}
