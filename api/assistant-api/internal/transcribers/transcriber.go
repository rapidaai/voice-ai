package internal_transcribes

import (
	"context"

	"github.com/rapidaai/pkg/utils"
)

// tokenizerrOption is a flexible map type that allows for various configuration options
// to be passed to a synthesizer. The string keys represent option names, and the interface{}
// values allow for any type of option value to be stored.
type TranscriberOptions struct {
	//
	Opts utils.Option

	//
	OnCompleteSentence func(
		ctx context.Context,
		contextId string,
		output string,
	) error
}

// tokenizerr is a generic interface that defines the contract for any type of synthesizer.
// It uses a generic type parameter IN to allow for different input types.
// The interface defines two methods: tokenizer and Flush.
type Transcriber interface {
	Transcribe(ctx context.Context, contextId string, text string, completed bool) error
}
