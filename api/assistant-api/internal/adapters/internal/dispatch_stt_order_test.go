package adapter_internal

import (
	"context"
	"testing"
	"time"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_router "github.com/rapidaai/api/assistant-api/internal/adapters/router"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

type orderedSpeechToTextTransformer struct {
	started      chan string
	releaseFirst chan struct{}
}

func (orderedSpeechToTextTransformer) Name() string { return "ordered" }

func (orderedSpeechToTextTransformer) Initialize() error { return nil }

func (transformer orderedSpeechToTextTransformer) Transform(_ context.Context, packet internal_type.Packet) error {
	audio, ok := packet.(internal_type.SpeechToTextAudioPacket)
	if !ok {
		return nil
	}

	chunk := string(audio.Audio)
	transformer.started <- chunk
	if chunk == "first" {
		<-transformer.releaseFirst
	}
	return nil
}

func (orderedSpeechToTextTransformer) Close(context.Context) error { return nil }

func TestInputDispatcher_PreservesSpeechToTextAudioOrder(t *testing.T) {
	channels := adapter_channel.NewRequestorChannels()
	transformer := orderedSpeechToTextTransformer{
		started:      make(chan string, 2),
		releaseFirst: make(chan struct{}),
	}
	requestor := &genericRequestor{
		channels:                channels,
		dispatchRoute:           adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), channels),
		speechToTextTransformer: transformer,
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go requestor.runInputDispatcher(ctx)

	if err := requestor.OnPacket(ctx,
		internal_type.SpeechToTextAudioPacket{ContextID: "ctx", Audio: []byte("first")},
		internal_type.SpeechToTextAudioPacket{ContextID: "ctx", Audio: []byte("second")},
	); err != nil {
		t.Fatalf("enqueue audio packets: %v", err)
	}

	select {
	case chunk := <-transformer.started:
		if chunk != "first" {
			t.Fatalf("expected first chunk to start first, got %q", chunk)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first audio chunk")
	}

	select {
	case chunk := <-transformer.started:
		t.Fatalf("audio chunk %q started before the first chunk completed", chunk)
	case <-time.After(50 * time.Millisecond):
	}

	close(transformer.releaseFirst)

	select {
	case chunk := <-transformer.started:
		if chunk != "second" {
			t.Fatalf("expected second chunk after first completed, got %q", chunk)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second audio chunk")
	}
}
