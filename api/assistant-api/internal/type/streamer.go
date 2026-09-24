// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_type

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// Streamer exchanges protobuf conversation messages with a channel transport.
type Streamer interface {
	// Context returns the stream's cancellation context.
	Context() context.Context

	// Recv waits for an incoming message or returns an error when the stream ends.
	Recv() (proto.Message, error)

	// Send delivers an outgoing message and reports transport errors.
	Send(proto.Message) error
}
