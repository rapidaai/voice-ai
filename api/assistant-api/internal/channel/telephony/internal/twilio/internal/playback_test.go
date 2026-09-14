package internal_twilio

import (
	"bytes"
	"testing"

	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
)

func TestMediaSessionPlaybackBufferedConversionPreservesResponseBoundaries(t *testing.T) {
	processor, err := NewAudioProcessor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer processor.inputWriter.Close()
	defer processor.outputWriter.Close()
	var completedIDs []string
	var sentAtCompletion []int
	sentFrames := 0
	session := internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		MediaEngine: processor,
		OutputSink: func(frame internal_telephony_media.AssistantOutputFrame) error {
			if len(frame.ProviderAudio) != OutputChunkSize {
				t.Fatalf("frame length = %d", len(frame.ProviderAudio))
			}
			sentFrames++
			return nil
		},
		StreamSink: func(message proto.Message) {
			if completion, ok := message.(*protos.ConversationPlaybackComplete); ok {
				completedIDs = append(completedIDs, completion.GetId())
				sentAtCompletion = append(sentAtCompletion, sentFrames)
			}
		},
	})
	defer session.Shutdown()
	// Short PCM chunks leave filter history and a partial transport frame.
	first := bytes.Repeat([]byte{1, 2}, 401)
	second := bytes.Repeat([]byte{3, 4}, 639)
	if _, err := session.HandleAssistantAudio("first", first, false); err != nil {
		t.Fatal(err)
	}
	for frame := session.NextFrame(); len(frame) > 0; frame = session.NextFrame() {
		if err := session.ConsumeFrame(frame); err != nil {
			t.Fatal(err)
		}
	}
	if len(completedIDs) != 0 {
		t.Fatal("streaming gap completed playback")
	}
	if _, err := session.HandleAssistantAudio("second", second, true); err != nil {
		t.Fatal(err)
	}
	if _, err := session.HandleAssistantAudio("first", nil, true); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 20; tick++ {
		if frame := session.NextFrame(); len(frame) > 0 {
			if err := session.ConsumeFrame(frame); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(completedIDs) != 2 || completedIDs[0] != "first" || completedIDs[1] != "second" {
		t.Fatalf("response completion order = %v", completedIDs)
	}
	if sentAtCompletion[0] != 2 || sentAtCompletion[1] != 4 {
		t.Fatalf("sent frames at completion = %v, want [2 4] including tails and padding", sentAtCompletion)
	}
}

func TestAudioProcessorClearDiscardsBufferedConversionTail(t *testing.T) {
	processor, err := NewAudioProcessor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer processor.inputWriter.Close()
	defer processor.outputWriter.Close()
	if err := processor.ProcessAssistantAudio(bytes.Repeat([]byte{1, 2}, 401), false); err != nil {
		t.Fatal(err)
	}
	processor.ClearOutputBuffer()
	if err := processor.ProcessAssistantAudio(nil, true); err != nil {
		t.Fatal(err)
	}
	if _, ok := processor.NextOutputFrame(); ok {
		t.Fatal("discarded response tail reappeared after flush")
	}
	if err := processor.ProcessAssistantAudio(make([]byte, BridgeOutputFrameSize), true); err != nil {
		t.Fatal(err)
	}
	frame, ok := processor.NextOutputFrame()
	if !ok || !bytes.Equal(frame.ProviderAudio, bytes.Repeat([]byte{MulawSilence}, OutputChunkSize)) {
		t.Fatal("new response contains discarded filter history")
	}
}

func TestMediaSessionPlaybackSameIDCannotReopenAfterTerminal(t *testing.T) {
	for _, queued := range []bool{false, true} {
		t.Run(map[bool]string{false: "after_drain", true: "before_drain"}[queued], func(t *testing.T) {
			processor, err := NewAudioProcessor(nil)
			if err != nil {
				t.Fatal(err)
			}
			defer processor.inputWriter.Close()
			defer processor.outputWriter.Close()
			var completions []*protos.ConversationPlaybackComplete
			var recorded []byte
			session := internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
				MediaEngine: processor,
				OutputSink: func(frame internal_telephony_media.AssistantOutputFrame) error {
					recorded = append(recorded, frame.BridgeAudio...)
					return nil
				},
				StreamSink: func(message proto.Message) {
					if completion, ok := message.(*protos.ConversationPlaybackComplete); ok {
						completions = append(completions, completion)
					}
				},
			})
			defer session.Shutdown()
			first := bytes.Repeat([]byte{1, 2}, BridgeOutputFrameSize/2)
			second := bytes.Repeat([]byte{3, 4}, BridgeOutputFrameSize/2)
			if accepted, err := session.HandleAssistantAudio("same-context", first, true); err != nil || !accepted {
				t.Fatalf("first message: accepted=%v error=%v", accepted, err)
			}
			if !queued {
				for tick := 0; tick < 4; tick++ {
					if frame := session.NextFrame(); len(frame) > 0 {
						if err := session.ConsumeFrame(frame); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			if accepted, err := session.HandleAssistantAudio("same-context", second, true); err != nil || accepted {
				t.Fatalf("same ID accepted=%v error=%v", accepted, err)
			}
			for tick := 0; tick < 6; tick++ {
				if frame := session.NextFrame(); len(frame) > 0 {
					if err := session.ConsumeFrame(frame); err != nil {
						t.Fatal(err)
					}
				}
			}
			if !bytes.Equal(recorded, first) {
				t.Fatal("same-ID audio reopened after terminal")
			}
			if len(completions) != 1 {
				t.Fatalf("completion count = %d, want 1", len(completions))
			}
			if completions[0].GetId() != "same-context" {
				t.Fatalf("first completion = %v", completions[0])
			}
		})
	}
}
