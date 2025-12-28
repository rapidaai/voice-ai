// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tokenizer

import (
	"context"
)

type TokenizerCallback func(
	ctx context.Context,
	contextId string,
	output string,
) error

// tokenizerr is a generic interface that defines the contract for any type of synthesizer.
// It uses a generic type parameter IN to allow for different input types.
// The interface defines two methods: tokenizer and Flush.
type Tokenizer interface {
	Tokenize(ctx context.Context, contextId string, text string, completed bool) error
}
