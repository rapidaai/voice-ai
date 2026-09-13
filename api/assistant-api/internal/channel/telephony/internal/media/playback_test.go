package internal_telephony_media

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
)

func TestMediaSessionInvalidPlaybackControlPreservesPendingOutput(t *testing.T) {
	for _, kind := range []protos.ConversationPlaybackControl_Kind{
		protos.ConversationPlaybackControl_KIND_UNSPECIFIED, protos.ConversationPlaybackControl_Kind(99),
	} {
		t.Run(kind.String(), func(t *testing.T) {
			engine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
			engine.outputFrames <- AssistantOutputFrame{ProviderAudio: []byte{1, 2}}
			var sent, providerClears int
			var completions []*protos.ConversationPlaybackComplete
			session := NewMediaSession(MediaSessionConfig{
				MediaEngine:       engine,
				OutputSink:        func(AssistantOutputFrame) error { sent++; return nil },
				SendProviderClear: func() error { providerClears++; return nil },
				StreamSink: func(message proto.Message) {
					if completion, ok := message.(*protos.ConversationPlaybackComplete); ok {
						completions = append(completions, completion)
					}
				},
			})
			defer session.Shutdown()
			invalid := &protos.ConversationPlaybackControl{Id: "response", Kind: kind}
			if handled, err := session.HandleOutputControl(invalid); !handled || err == nil {
				t.Fatalf("invalid control: handled=%v, error=%v", handled, err)
			}
			if session.outputPaused || session.outputFlushed || sent != 0 || providerClears != 0 || engine.clearCount.Load() != 0 {
				t.Fatal("invalid control changed output state or sent output")
			}
			if accepted, err := session.HandleAssistantAudio("response", []byte{1, 2}, true); !accepted || err != nil {
				t.Fatalf("assistant audio: accepted=%v, error=%v", accepted, err)
			}
			frame := session.NextFrame()
			if !bytes.Equal(frame, []byte{1, 2}) {
				t.Fatalf("unexpected frame: %v", frame)
			}
			if handled, err := session.HandleOutputControl(&protos.ConversationPlaybackControl{Id: "response", Kind: protos.ConversationPlaybackControl_PAUSE}); !handled || err != nil {
				t.Fatalf("pause: handled=%v, error=%v", handled, err)
			}
			if handled, err := session.HandleOutputControl(invalid); !handled || err == nil {
				t.Fatalf("invalid paused control: handled=%v, error=%v", handled, err)
			}
			if !session.outputPaused || session.outputFlushed || sent != 0 || providerClears != 0 || engine.clearCount.Load() != 0 || len(completions) != 0 {
				t.Fatal("invalid control changed paused output or emitted a completion")
			}
			if len(session.NextFrame()) != 0 {
				t.Fatal("invalid control resumed paused output")
			}
			if handled, err := session.HandleOutputControl(&protos.ConversationPlaybackControl{Id: "response", Kind: protos.ConversationPlaybackControl_CONTINUE}); !handled || err != nil {
				t.Fatalf("continue: handled=%v, error=%v", handled, err)
			}
			if resumed := session.NextFrame(); !bytes.Equal(resumed, frame) {
				t.Fatalf("pending frame changed: %v", resumed)
			}
			if err := session.ConsumeFrame(frame); err != nil {
				t.Fatal(err)
			}
			_ = session.NextFrame()
			if sent != 1 || len(completions) != 1 || completions[0].GetId() != "response" || completions[0].GetTime() == nil {
				t.Fatalf("valid continue failed to complete playback: sent=%d, completions=%v", sent, completions)
			}
			if handled, err := session.HandleOutputControl(&protos.ConversationPlaybackControl{Id: "response", Kind: protos.ConversationPlaybackControl_FLUSH}); !handled || err != nil {
				t.Fatalf("flush: handled=%v, error=%v", handled, err)
			}
			if providerClears != 1 || engine.clearCount.Load() != 1 {
				t.Fatal("valid flush must clear local and provider output exactly once")
			}
		})
	}
}

func TestMediaSessionPlaybackRequiresTerminalAndSuccessfulSend(t *testing.T) {
	for _, scenario := range []string{"success", "streaming_gap", "fetched", "paused", "continued", "flushed", "failed", "no_sink", "closed", "cancelled", "conversion_failed"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			engine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
			engine.outputFrames <- AssistantOutputFrame{ProviderAudio: []byte{1}}
			var completions []*protos.ConversationPlaybackComplete
			sink := func(AssistantOutputFrame) error {
				if scenario == "failed" {
					return errors.New("transport write failed")
				}
				return nil
			}
			if scenario == "no_sink" {
				sink = nil
			}
			if scenario == "conversion_failed" {
				engine.assistantErr = errors.New("conversion failed")
			}
			session := NewMediaSession(MediaSessionConfig{
				Context: ctx, MediaEngine: engine, OutputSink: sink,
				StreamSink: func(message proto.Message) {
					if completion, ok := message.(*protos.ConversationPlaybackComplete); ok {
						completions = append(completions, completion)
					}
				},
			})
			defer session.Shutdown()
			_, err := session.HandleAssistantAudio("response", []byte{1}, scenario != "streaming_gap")
			if (err != nil) != (scenario == "conversion_failed") {
				t.Fatalf("unexpected conversion result: %v", err)
			}
			frame := session.NextFrame()
			if len(frame) == 0 || len(completions) != 0 {
				t.Fatal("fetch must return audio without completing playback")
			}
			switch scenario {
			case "paused", "continued":
				_, _ = session.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_PAUSE})
			case "flushed":
				_, _ = session.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH})
			case "closed":
				session.Shutdown()
			case "cancelled":
				cancel()
			}
			if scenario != "fetched" {
				_ = session.ConsumeFrame(frame)
			}
			if scenario == "continued" {
				_, _ = session.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_CONTINUE})
				_ = session.ConsumeFrame(session.NextFrame())
			}
			_ = session.NextFrame()
			wantCompletion := scenario == "success" || scenario == "continued"
			if (len(completions) == 1) != wantCompletion {
				t.Fatalf("completion count = %d, want completion = %v", len(completions), wantCompletion)
			}
			if wantCompletion {
				if completions[0].GetId() != "response" || completions[0].GetTime() == nil {
					t.Fatalf("unexpected completion: %v", completions[0])
				}
				_ = session.NextFrame()
				if len(completions) != 1 {
					t.Fatal("completion repeated")
				}
			}
		})
	}
}

func TestMediaSessionPlaybackCompletionDoesNotHoldControlLock(t *testing.T) {
	engine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
	engine.outputFrames <- AssistantOutputFrame{ProviderAudio: []byte{1}}
	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	session := NewMediaSession(MediaSessionConfig{
		MediaEngine: engine,
		OutputSink:  func(AssistantOutputFrame) error { return nil },
		StreamSink: func(message proto.Message) {
			if _, ok := message.(*protos.ConversationPlaybackComplete); ok {
				close(entered)
				<-release
			}
		},
	})
	defer session.Shutdown()
	_, _ = session.HandleAssistantAudio("response", []byte{1}, true)
	_ = session.ConsumeFrame(session.NextFrame())
	go func() {
		defer close(finished)
		session.NextFrame()
	}()
	defer func() { close(release); <-finished }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("completion sink was not reached")
	}
	controlled := make(chan struct{})
	go func() {
		defer close(controlled)
		_, _ = session.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_PAUSE})
		_, _ = session.HandleOutputControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH})
	}()
	select {
	case <-controlled:
	case <-time.After(time.Second):
		t.Fatal("completion sink blocked output controls")
	}
}

func TestMediaSessionPlaybackCompletionClosesMessageIDOnce(t *testing.T) {
	engine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 1)}
	engine.outputFrames <- AssistantOutputFrame{ProviderAudio: []byte{1}}
	var completions []*protos.ConversationPlaybackComplete
	session := NewMediaSession(MediaSessionConfig{
		MediaEngine: engine,
		OutputSink:  func(AssistantOutputFrame) error { return nil },
		StreamSink: func(message proto.Message) {
			if completion, ok := message.(*protos.ConversationPlaybackComplete); ok {
				completions = append(completions, completion)
			}
		},
	})
	defer session.Shutdown()

	accepted, err := session.HandleAssistantAudio("response", []byte{1}, true)
	if err != nil {
		t.Fatalf("first message: %v", err)
	}
	if !accepted {
		t.Fatal("first message was rejected")
	}
	accepted, err = session.HandleAssistantAudio("response", []byte{2}, true)
	if err != nil || accepted {
		t.Fatalf("same ID before drain accepted=%v error=%v", accepted, err)
	}
	if err := session.ConsumeFrame(session.NextFrame()); err != nil {
		t.Fatalf("consume frame: %v", err)
	}
	_ = session.NextFrame()
	accepted, err = session.HandleAssistantAudio("response", []byte{2}, true)
	if err != nil || accepted {
		t.Fatalf("same ID after drain accepted=%v error=%v", accepted, err)
	}
	engine.outputFrames <- AssistantOutputFrame{ProviderAudio: []byte{2}}
	accepted, err = session.HandleAssistantAudio("response-2", []byte{2}, true)
	if err != nil {
		t.Fatalf("second message: %v", err)
	}
	if !accepted {
		t.Fatal("second message was rejected")
	}
	if err := session.ConsumeFrame(session.NextFrame()); err != nil {
		t.Fatalf("consume second frame: %v", err)
	}
	_ = session.NextFrame()

	if len(completions) != 2 {
		t.Fatalf("completion count = %d, want 2", len(completions))
	}
	if completions[0].GetId() != "response" {
		t.Fatalf("first completion = %v", completions[0])
	}
	if completions[1].GetId() != "response-2" {
		t.Fatalf("second completion = %v", completions[1])
	}
}

func TestMediaSessionDiscardForTransferSuppressesSameIDCompletion(t *testing.T) {
	engine := &fakeMediaEngine{outputFrames: make(chan AssistantOutputFrame, 2)}
	engine.outputFrames <- AssistantOutputFrame{ProviderAudio: []byte{1}}
	engine.outputFrames <- AssistantOutputFrame{ProviderAudio: []byte{2}}
	var completions []*protos.ConversationPlaybackComplete
	session := NewMediaSession(MediaSessionConfig{
		MediaEngine: engine,
		OutputSink:  func(AssistantOutputFrame) error { return nil },
		StreamSink: func(message proto.Message) {
			if completion, ok := message.(*protos.ConversationPlaybackComplete); ok {
				completions = append(completions, completion)
			}
		},
	})
	defer session.Shutdown()

	accepted, err := session.HandleAssistantAudio("transfer-response", []byte{1}, false)
	if err != nil || !accepted {
		t.Fatalf("initial response accepted=%t err=%v", accepted, err)
	}
	if err := session.ConsumeFrame(session.NextFrame()); err != nil {
		t.Fatalf("consume initial frame: %v", err)
	}
	session.DiscardForTransfer()

	accepted, err = session.HandleAssistantAudio("transfer-response", []byte{2}, true)
	if err != nil || !accepted {
		t.Fatalf("same ID resumed response accepted=%t err=%v", accepted, err)
	}
	if err := session.ConsumeFrame(session.NextFrame()); err != nil {
		t.Fatalf("consume resumed frame: %v", err)
	}
	_ = session.NextFrame()
	if len(completions) != 0 {
		t.Fatalf("completion count = %d, want 0 for transfer-truncated response", len(completions))
	}
}
