package internal_adapter_request_streamers

import (
	"context"

	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	lexatic_backend "github.com/rapidaai/protos"
)

type StreamConfig struct {
	Audio *internal_voices.AudioConfig `json:"audio,omitempty"`
	Text  *struct {
		Charset string `json:"charset"` // Character set (e.g., UTF-8)
	} `json:"text,omitempty"`
}

// StreamAttribute represents the configuration with appropriate JSON tags.
type StreamAttribute struct {
	// Input audio configuration
	InputConfig *StreamConfig `json:"input_config,omitempty"`

	// Output audio configuration
	OutputConfig *StreamConfig `json:"output_config,omitempty"`
}

type Streamer interface {
	// Context returns the context associated with the stream.
	// This context is typically used to control cancellation and deadlines.
	Context() context.Context

	// Recv receives the next input value from the stream.
	// It returns the received value and any error encountered.
	// If the stream is closed, it should return (zero value, io.EOF).
	Recv() (*lexatic_backend.AssistantMessagingRequest, error)

	// Send sends an output value to the stream.
	// It returns an error if the send operation fails.
	Send(*lexatic_backend.AssistantMessagingResponse) error

	// config of streamer
	// later few more things can be added that allow more customization for source
	Config() *StreamAttribute
}
