package lifecycle

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMessageLifecycle_DefaultsMode(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("ctx"), WithMode(type_enums.MessageMode("")))
	if got := l.Mode(); got != type_enums.TextMode {
		t.Fatalf("unexpected mode, got=%v want=%v", got, type_enums.TextMode)
	}
	if got := l.State(); got != MessageStateAssistantIdle {
		t.Fatalf("unexpected state, got=%s want=%s", got, MessageStateAssistantIdle)
	}
}

func TestMessageLifecycle_AcceptUserTurnReplacesAssistantMessage(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("ctx-old"), WithMode(type_enums.MessageMode("")))
	if err := l.OnGenerationStarted("ctx-old"); err != nil {
		t.Fatalf("unexpected assistant generating error: %v", err)
	}
	turn, err := l.OnUserTurnStarted("ctx-old", string(internal_type.PacketNameUserTextReceived), "text", "new request")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if turn.PreviousContextID != "ctx-old" {
		t.Fatalf("unexpected old context id: %s", turn.PreviousContextID)
	}
	if turn.ContextID == "" || turn.ContextID == "ctx-old" {
		t.Fatalf("unexpected new context id: %s", turn.ContextID)
	}
	if got := l.ContextID(); got != turn.ContextID {
		t.Fatalf("unexpected context id, got=%s want=%s", got, turn.ContextID)
	}
	if got := l.State(); got != MessageStateAssistantIdle {
		t.Fatalf("unexpected state, got=%s want=%s", got, MessageStateAssistantIdle)
	}
}

func TestMessageLifecycle_AcceptUserTurnPreservesActiveSpeech(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("ctx-old"), WithMode(type_enums.TextMode))
	l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-old", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}, internal_options.BargeInTriggerVAD)
	contextID := l.ContextID()
	turn, err := l.OnUserTurnStarted(contextID, string(internal_type.PacketNameUserTextReceived), "text", "new request")
	if err != nil {
		t.Fatal(err)
	}
	if turn.ContextID != contextID || turn.PreviousContextID != "" || l.ContextID() != contextID {
		t.Fatalf("active user speech was replaced: %+v", turn)
	}
	if got := l.State(); got != MessageStateUserSpeaking {
		t.Fatalf("unexpected state: %s", got)
	}
}

func TestMessageLifecycle_UserFlow(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("ctx"), WithMode(type_enums.TextMode))
	l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "ctx", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}, internal_options.BargeInTriggerVAD)
	l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: l.ContextID(), Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}, internal_options.BargeInTriggerVAD)
	if err := l.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: l.ContextID(), Speech: "hello"}); err != nil {
		t.Fatalf("unexpected user finished error: %v", err)
	}
	if got := l.State(); got != MessageStateUserFinished {
		t.Fatalf("unexpected state, got=%s want=%s", got, MessageStateUserFinished)
	}
}

func TestMessageLifecycle_CompleteUserSpeechRejectsStaleAndEmptySpeech(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("ctx"), WithMode(type_enums.TextMode))
	if _, err := l.OnTranscriptReceived("ctx", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := l.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: "old", Speech: "hello"}); !errors.Is(err, ErrStaleContext) {
		t.Fatalf("expected stale speech to fail, got=%v", err)
	}
	if err := l.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: "ctx", Speech: "  "}); err != nil {
		t.Fatal(err)
	}
	if l.State() != MessageStateUserListening {
		t.Fatalf("empty or stale speech completed the user turn: %s", l.State())
	}
}

func TestMessageLifecycle_AssistantFlow(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("ctx"), WithMode(type_enums.TextMode), WithSend(func(proto.Message) error { return nil }))
	if err := l.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: "ctx", Speech: "hello"}); err != nil {
		t.Fatalf("unexpected user finished error: %v", err)
	}
	if err := l.OnGenerationStarted("ctx"); err != nil {
		t.Fatalf("unexpected assistant generating error: %v", err)
	}
	if err := l.OnSpeechStarted("ctx"); err != nil {
		t.Fatalf("unexpected assistant speaking error: %v", err)
	}
	l.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "ctx", Text: "answer"})
	if l.CanStartIdleTimeout("ctx") {
		t.Fatal("generation completion must not start idle timeout before delivery")
	}
	if err := l.SendAssistantMessage(&protos.ConversationAssistantMessage{
		Id: "ctx", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "answer"},
	}); err != nil {
		t.Fatal(err)
	}
	if got := l.State(); got != MessageStateAssistantIdle {
		t.Fatalf("unexpected state, got=%s want=%s", got, MessageStateAssistantIdle)
	}
}

func TestMessageLifecycle_UnclearPromptRejectsDuplicate(t *testing.T) {
	text := "Could you repeat that?"
	l := NewMessageLifecycle(WithContextID("ctx"), WithMode(type_enums.TextMode), WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
		return &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputMessage: &text}, nil
	}))
	require.NoError(t, l.Initialize(context.Background()))
	t.Cleanup(l.StopUnclearInput)
	if _, err := l.OnTranscriptReceived("ctx", "hello"); err != nil {
		t.Fatal(err)
	}
	turn, prompt, err := l.OnPrompt(internal_type.UnclearInputExpiredPacket{ContextID: "ctx"})
	if err != nil || prompt.Text != text || prompt.ContextID != turn.ContextID || turn.ContextID == "ctx" {
		t.Fatalf("unexpected prompt: turn=%+v prompt=%+v err=%v", turn, prompt, err)
	}
	if _, _, err := l.OnPrompt(internal_type.UnclearInputExpiredPacket{ContextID: "ctx"}); !errors.Is(err, ErrStaleContext) {
		t.Fatalf("expected duplicate prompt to fail, got=%v", err)
	}
}

func TestMessageLifecycle_IdlePromptRejectsDuplicate(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("ctx"), WithMode(type_enums.TextMode))
	turn, prompt, err := l.OnPrompt(internal_type.IdleTimeoutExpiredPacket{ContextID: "ctx"})
	if err != nil || turn.PreviousContextID != "ctx" || prompt.ContextID != turn.ContextID || turn.ContextID == "ctx" {
		t.Fatalf("unexpected prompt: turn=%+v prompt=%+v err=%v", turn, prompt, err)
	}
	if _, _, err := l.OnPrompt(internal_type.IdleTimeoutExpiredPacket{ContextID: "ctx"}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected duplicate prompt to fail, got=%v", err)
	}
}

func TestMessageLifecycle_StaleContextRejected(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("ctx"), WithMode(type_enums.TextMode))
	if err := l.OnGenerationStarted("old"); !errors.Is(err, ErrStaleContext) {
		t.Fatalf("expected stale context error, got=%v", err)
	}
}

func TestMessageLifecycle_ObservePlaybackCompletion(t *testing.T) {
	lifecycle := NewMessageLifecycle(WithContextID("response-1"), WithMode(type_enums.AudioMode), WithSend(func(proto.Message) error { return nil }))
	t.Cleanup(func() { lifecycle.OnMessageFailed(lifecycle.ContextID()) })
	if err := lifecycle.OnGenerationStarted("response-1"); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.OnSpeechStarted("response-1"); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		name      string
		contextID string
		issued    bool
		wantError error
	}{
		{name: "empty", wantError: ErrEmptyContextID},
		{name: "stale", contextID: "response-old", wantError: ErrStaleContext},
		{name: "early", contextID: "response-1", wantError: ErrPlaybackTerminalNotIssued},
		{name: "accepted", contextID: "response-1", issued: true},
		{name: "duplicate", contextID: "response-1", wantError: ErrDuplicatePlaybackCompletion},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if scenario.issued {
				lifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: scenario.contextID, Text: "answer"})
				if err := lifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: scenario.contextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}); err != nil {
					t.Fatal(err)
				}
			}
			if err := lifecycle.OnPlaybackCompleted(scenario.contextID); !errors.Is(err, scenario.wantError) {
				t.Fatalf("unexpected receipt result: got %v, want %v", err, scenario.wantError)
			}
			if lifecycle.State() != MessageStateAssistantSpeaking {
				t.Fatalf("receipt changed lifecycle state: %s", lifecycle.State())
			}
		})
	}
	turn, err := lifecycle.OnUserTurnStarted("response-1", string(internal_type.PacketNameUserTextReceived), "text", "next request")
	if err != nil {
		t.Fatal(err)
	}
	nextContextID := turn.ContextID
	if err := lifecycle.OnPlaybackCompleted("response-1"); !errors.Is(err, ErrStaleContext) {
		t.Fatalf("old receipt after rotation: %v", err)
	}
	if err := lifecycle.OnPlaybackCompleted(nextContextID); !errors.Is(err, ErrPlaybackTerminalNotIssued) {
		t.Fatalf("rotation must clear terminal eligibility: %v", err)
	}
	lifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: nextContextID, Text: "answer"})
	if err := lifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: nextContextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.OnPlaybackCompleted(nextContextID); err != nil {
		t.Fatalf("new response receipt rejected: %v", err)
	}
	if lifecycle.State() != MessageStateAssistantIdle {
		t.Fatalf("receipt changed idle state: %s", lifecycle.State())
	}
}

func TestMessageLifecycle_ConcurrentPlaybackCompletionAcceptsOnce(t *testing.T) {
	lifecycle := NewMessageLifecycle(WithContextID("response-1"), WithMode(type_enums.AudioMode), WithSend(func(proto.Message) error { return nil }))
	t.Cleanup(func() { lifecycle.OnMessageFailed("response-1") })
	lifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "response-1", Text: "answer"})
	if err := lifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "response-1", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}); err != nil {
		t.Fatal(err)
	}
	var accepted atomic.Int32
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			err := lifecycle.OnPlaybackCompleted("response-1")
			if err == nil {
				accepted.Add(1)
			} else if !errors.Is(err, ErrDuplicatePlaybackCompletion) {
				t.Errorf("unexpected receipt failure: %v", err)
			}
		})
	}
	workers.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("expected one accepted receipt, got %d", accepted.Load())
	}
}

func TestMessageLifecycle_PlaybackTerminalRevocationIsResponseScoped(t *testing.T) {
	lifecycle := NewMessageLifecycle(WithContextID("response-1"), WithMode(type_enums.AudioMode), WithSend(func(proto.Message) error { return nil }))
	t.Cleanup(func() { lifecycle.OnMessageFailed(lifecycle.ContextID()) })
	lifecycle.OnMessageFailed("response-1")
	if err := lifecycle.OnPlaybackCompleted("response-1"); !errors.Is(err, ErrPlaybackTerminalNotIssued) {
		t.Fatalf("revoked terminal must reject receipt: %v", err)
	}
	turn, err := lifecycle.OnUserTurnStarted("response-1", string(internal_type.PacketNameUserTextReceived), "text", "next request")
	if err != nil {
		t.Fatal(err)
	}
	nextContextID := turn.ContextID
	lifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: nextContextID, Text: "answer"})
	if err := lifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: nextContextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}); err != nil {
		t.Fatal(err)
	}
	lifecycle.OnMessageFailed("response-1")
	if err := lifecycle.OnPlaybackCompleted(nextContextID); err != nil {
		t.Fatalf("stale revocation affected current terminal: %v", err)
	}
}

func TestMessageLifecycle_OnGenerationCompleted(t *testing.T) {
	for _, phase := range []string{"generating", "speaking"} {
		t.Run(phase, func(t *testing.T) {
			message := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode))
			require.NoError(t, message.OnGenerationStarted("message"))
			if phase == "speaking" {
				require.NoError(t, message.OnSpeechStarted("message"))
			}
			assert.Empty(t, message.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "stale", Text: "old"}))
			require.NoError(t, message.OnGenerationStarted("message"), "stale completion must not close current generation")
			completed := internal_type.LLMResponseDonePacket{ContextID: "message", Text: "answer"}
			assert.Equal(t, []internal_type.Packet{internal_type.MessageCreatePacket{
				ContextID: "message", MessageRole: "assistant", Text: "answer",
			}}, message.OnGenerationCompleted(completed))
			if phase == "speaking" {
				assert.Equal(t, MessageStateAssistantSpeaking, message.State())
			} else {
				assert.Equal(t, MessageStateAssistantGenerated, message.State())
			}
			assert.ErrorIs(t, message.OnGenerationStarted("message"), ErrInvalidTransition)
			assert.False(t, message.CanStartIdleTimeout("message"))
			assert.Empty(t, message.OnGenerationCompleted(completed), "duplicate completion must not repeat message creation")
		})
	}

	t.Run("failed message", func(t *testing.T) {
		message := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode))
		require.NoError(t, message.OnGenerationStarted("message"))
		message.OnMessageFailed("message")
		assert.Empty(t, message.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "message", Text: "answer"}))
		assert.False(t, message.CanStartIdleTimeout("message"))
	})
}

func TestMessageLifecycle_CloseStopsPendingWork(t *testing.T) {
	for _, phase := range []string{"playback", "pause", "unclear input"} {
		t.Run(phase, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				packets := make(chan internal_type.Packet, 16)
				timeout := 1.0
				message := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode),
					WithInterruption(phase == "pause"),
					WithSend(func(proto.Message) error { return nil }),
					WithOnPacket(func(emitted ...internal_type.Packet) error {
						for _, packet := range emitted {
							if phase == "unclear input" {
								if _, expired := packet.(internal_type.UnclearInputExpiredPacket); !expired {
									continue
								}
							}
							packets <- packet
						}
						return nil
					}),
					WithInterruptionExpiry(func(expired internal_type.InterruptionDecisionExpiredPacket) { packets <- expired }),
					WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
						return &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputTimeout: &timeout}, nil
					}))
				if phase == "unclear input" {
					require.NoError(t, message.Initialize(context.Background()))
					message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
						ContextID: message.ContextID(), Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
					}, internal_options.BargeInTriggerVAD)
					message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
						ContextID: message.ContextID(), Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
					}, internal_options.BargeInTriggerVAD)
				} else {
					require.NoError(t, message.OnGenerationStarted("message"))
					message.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "message", Text: "answer"})
					require.NoError(t, message.SendAssistantMessage(&protos.ConversationAssistantMessage{
						Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "answer"},
					}))
					require.NoError(t, message.SendAssistantMessage(&protos.ConversationAssistantMessage{
						Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
					}))
					if phase == "pause" {
						decision := message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
							ContextID: "message", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
						}, internal_options.BargeInTriggerVAD)
						require.NotNil(t, decision.Pause)
						require.NoError(t, message.SendPlaybackControl(&protos.ConversationPlaybackPause{Id: "message"}))
						assert.Nil(t, message.OnPlaybackPaused(*decision.Pause, nil))
					}
				}
				control := message.Close(message.ContextID())
				if phase == "pause" {
					require.NotNil(t, control)
					assert.Equal(t, "message", control.Id)
					require.NoError(t, message.SendPlaybackControl(control))
				} else {
					assert.Nil(t, control)
				}
				assert.Nil(t, message.Close(message.ContextID()))
				time.Sleep(10 * time.Second)
				synctest.Wait()
				assert.Empty(t, packets, "closed message must not emit timeout or completion events")
				assert.ErrorIs(t, message.OnPlaybackCompleted(message.ContextID()), ErrPlaybackTerminalNotIssued)
				assert.ErrorIs(t, message.OnGenerationStarted(message.ContextID()), ErrInvalidTransition)
			})
		})
	}
}
