package internal_streamers

import (
	"context"

	"github.com/rapidaai/protos"
)

type Streamer interface {
	// Context returns the context associated with the stream.
	// This context is typically used to control cancellation and deadlines.
	Context() context.Context

	// Recv receives the next input value from the stream.
	// It returns the received value and any error encountered.
	// If the stream is closed, it should return (zero value, io.EOF).
	Recv() (*protos.AssistantMessagingRequest, error)

	// Send sends an output value to the stream.
	// It returns an error if the send operation fails.
	Send(*protos.AssistantMessagingResponse) error
}
