package adapter_internal

import (
	"context"
	"testing"
	"time"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	adapter_router "github.com/rapidaai/api/assistant-api/internal/adapters/router"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
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

type orderedEOSExecutor struct {
	started      chan string
	releaseAudio chan struct{}
}

func (orderedEOSExecutor) Name() string { return "ordered-eos" }

func (orderedEOSExecutor) Options() utils.Option { return nil }

func (orderedEOSExecutor) Arguments() (map[string]string, error) { return nil, nil }

func (executor orderedEOSExecutor) Execute(_ context.Context, packet internal_type.Packet) error {
	switch packet.(type) {
	case internal_type.EndOfSpeechAudioPacket:
		executor.started <- "audio"
		<-executor.releaseAudio
	case internal_type.SpeechToTextPacket:
		executor.started <- "stt"
	}
	return nil
}

func (orderedEOSExecutor) Close(context.Context) error { return nil }

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

func TestInputDispatcher_PreservesEndOfSpeechAudioBeforeTranscript(t *testing.T) {
	channels := adapter_channel.NewRequestorChannels()
	streamer := &streamTestStreamer{}
	executor := orderedEOSExecutor{
		started:      make(chan string, 2),
		releaseAudio: make(chan struct{}),
	}
	requestor := &genericRequestor{
		channels:            channels,
		dispatchRoute:       adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), channels),
		streamer:            streamer,
		messageLifecycle:    adapter_lifecycle.NewMessageLifecycle(adapter_lifecycle.WithContextID("ctx"), adapter_lifecycle.WithSend(streamer.Send)),
		endOfSpeechExecutor: executor,
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go requestor.runInputDispatcher(ctx)

	if err := requestor.OnPacket(ctx,
		internal_type.EndOfSpeechAudioPacket{ContextID: "ctx", Audio: []byte("audio")},
		internal_type.SpeechToTextPacket{ContextID: "ctx", Script: "hello", Interim: true},
	); err != nil {
		t.Fatalf("enqueue packets: %v", err)
	}

	select {
	case event := <-executor.started:
		if event != "audio" {
			t.Fatalf("expected EOS audio to start first, got %q", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for EOS audio")
	}

	select {
	case event := <-executor.started:
		t.Fatalf("transcript reached EOS before audio append completed: %q", event)
	case <-time.After(50 * time.Millisecond):
	}

	close(executor.releaseAudio)

	select {
	case event := <-executor.started:
		if event != "stt" {
			t.Fatalf("expected STT after EOS audio completed, got %q", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for STT")
	}
}
