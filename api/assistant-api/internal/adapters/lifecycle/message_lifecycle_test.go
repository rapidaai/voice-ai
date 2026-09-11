package lifecycle

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
)

func TestMessageLifecycle_DefaultsMode(t *testing.T) {
	l := NewMessageLifecycleWithContext("ctx", type_enums.MessageMode(""))
	if got := l.Mode(); got != type_enums.TextMode {
		t.Fatalf("unexpected mode, got=%v want=%v", got, type_enums.TextMode)
	}
	if got := l.State(); got != MessageStateAssistantIdle {
		t.Fatalf("unexpected state, got=%s want=%s", got, MessageStateAssistantIdle)
	}
}

func TestMessageLifecycle_RotateContext(t *testing.T) {
	l := NewMessageLifecycleWithContext("ctx-old", type_enums.MessageMode(""))
	if err := l.AssistantGenerating("ctx-old"); err != nil {
		t.Fatalf("unexpected assistant generating error: %v", err)
	}
	oldContextID, newContextID, err := l.RotateContext()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if oldContextID != "ctx-old" {
		t.Fatalf("unexpected old context id, got=%s want=ctx-old", oldContextID)
	}
	if newContextID == "" || newContextID == "ctx-old" {
		t.Fatalf("unexpected new context id, got=%s", newContextID)
	}
	if got := l.ContextID(); got != newContextID {
		t.Fatalf("unexpected context id, got=%s want=%s", got, newContextID)
	}
	if got := l.State(); got != MessageStateAssistantIdle {
		t.Fatalf("unexpected state, got=%s want=%s", got, MessageStateAssistantIdle)
	}
}

func TestMessageLifecycle_RotateContextAfterUserSpeakingResetsState(t *testing.T) {
	l := NewMessageLifecycleWithContext("ctx-old", type_enums.TextMode)
	if err := l.UserSpeaking("ctx-old"); err != nil {
		t.Fatalf("unexpected user speaking error: %v", err)
	}
	oldContextID, newContextID, err := l.RotateContext()
	if err != nil {
		t.Fatalf("unexpected rotate context error: %v", err)
	}
	if oldContextID != "ctx-old" {
		t.Fatalf("unexpected old context id, got=%s want=ctx-old", oldContextID)
	}
	if newContextID == "" || newContextID == "ctx-old" {
		t.Fatalf("unexpected new context id, got=%s", newContextID)
	}
	if got := l.ContextID(); got != newContextID {
		t.Fatalf("unexpected current context id, got=%s want=%s", got, newContextID)
	}
	if got := l.State(); got != MessageStateAssistantIdle {
		t.Fatalf("unexpected state, got=%s want=%s", got, MessageStateAssistantIdle)
	}
}

func TestMessageLifecycle_UserFlow(t *testing.T) {
	l := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
	if err := l.UserIdle("ctx"); err != nil {
		t.Fatalf("unexpected user idle error: %v", err)
	}
	if err := l.UserListening("ctx"); err != nil {
		t.Fatalf("unexpected user listening error: %v", err)
	}
	if err := l.UserSpeaking("ctx"); err != nil {
		t.Fatalf("unexpected user speaking error: %v", err)
	}
	if err := l.UserThinking("ctx"); err != nil {
		t.Fatalf("unexpected user thinking error: %v", err)
	}
	if err := l.UserFinished("ctx"); err != nil {
		t.Fatalf("unexpected user finished error: %v", err)
	}
	if got := l.State(); got != MessageStateUserFinished {
		t.Fatalf("unexpected state, got=%s want=%s", got, MessageStateUserFinished)
	}
}

func TestMessageLifecycle_UserFinishedRejectsUserSpeaking(t *testing.T) {
	l := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
	if err := l.UserSpeaking("ctx"); err != nil {
		t.Fatalf("unexpected user speaking error: %v", err)
	}
	if err := l.UserFinished("ctx"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected user speaking finish to fail, got=%v", err)
	}
	if err := l.UserListening("ctx"); err != nil {
		t.Fatalf("unexpected user listening error: %v", err)
	}
	if err := l.UserFinished("ctx"); err != nil {
		t.Fatalf("unexpected user finished error: %v", err)
	}
}

func TestMessageLifecycle_AssistantFlow(t *testing.T) {
	l := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
	if err := l.UserFinished("ctx"); err != nil {
		t.Fatalf("unexpected user finished error: %v", err)
	}
	if err := l.AssistantGenerating("ctx"); err != nil {
		t.Fatalf("unexpected assistant generating error: %v", err)
	}
	if err := l.AssistantGenerated("ctx"); err != nil {
		t.Fatalf("unexpected assistant generated error: %v", err)
	}
	if err := l.AssistantSpeaking("ctx"); err != nil {
		t.Fatalf("unexpected assistant speaking error: %v", err)
	}
	if err := l.AssistantFinished("ctx"); err != nil {
		t.Fatalf("unexpected assistant finished error: %v", err)
	}
	if err := l.AssistantIdle("ctx"); err != nil {
		t.Fatalf("unexpected assistant idle error: %v", err)
	}
	if got := l.State(); got != MessageStateAssistantIdle {
		t.Fatalf("unexpected state, got=%s want=%s", got, MessageStateAssistantIdle)
	}
}

func TestMessageLifecycle_UserPromptedCountsAndBlocksDuplicate(t *testing.T) {
	l := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
	if err := l.UserListening("ctx"); err != nil {
		t.Fatalf("unexpected user listening error: %v", err)
	}
	if err := l.UserPrompted("ctx"); err != nil {
		t.Fatalf("unexpected user prompted error: %v", err)
	}
	if got := l.UserPromptCount(); got != 1 {
		t.Fatalf("unexpected user prompt count, got=%d want=1", got)
	}
	if err := l.UserPrompted("ctx"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected duplicate user prompted to fail, got=%v", err)
	}
	if got := l.UserPromptCount(); got != 1 {
		t.Fatalf("unexpected user prompt count, got=%d want=1", got)
	}
}

func TestMessageLifecycle_AssistantPromptedCountsAndBlocksDuplicate(t *testing.T) {
	l := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
	if err := l.AssistantPrompted("ctx"); err != nil {
		t.Fatalf("unexpected assistant prompted error: %v", err)
	}
	if got := l.AssistantPromptCount(); got != 1 {
		t.Fatalf("unexpected assistant prompt count, got=%d want=1", got)
	}
	if err := l.AssistantPrompted("ctx"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected duplicate assistant prompted to fail, got=%v", err)
	}
	if got := l.AssistantPromptCount(); got != 1 {
		t.Fatalf("unexpected assistant prompt count, got=%d want=1", got)
	}
}

func TestMessageLifecycle_StaleContextRejected(t *testing.T) {
	l := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
	if err := l.AssistantGenerating("old"); !errors.Is(err, ErrStaleContext) {
		t.Fatalf("expected stale context error, got=%v", err)
	}
}

func TestMessageLifecycle_ObservePlaybackCompletion(t *testing.T) {
	lifecycle := NewMessageLifecycleWithContext("response-1", type_enums.AudioMode)
	t.Cleanup(func() { lifecycle.FailAssistantMessage(lifecycle.ContextID()) })
	if err := lifecycle.AssistantGenerating("response-1"); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.AssistantSpeaking("response-1"); err != nil {
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
				lifecycle.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: scenario.contextID, Text: "answer"})
				if err := lifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: scenario.contextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}, func(proto.Message) error { return nil }); err != nil {
					t.Fatal(err)
				}
			}
			if err := lifecycle.ObservePlaybackCompletion(scenario.contextID); !errors.Is(err, scenario.wantError) {
				t.Fatalf("unexpected receipt result: got %v, want %v", err, scenario.wantError)
			}
			if lifecycle.State() != MessageStateAssistantSpeaking {
				t.Fatalf("receipt changed lifecycle state: %s", lifecycle.State())
			}
		})
	}
	_, nextContextID, err := lifecycle.RotateContext()
	if err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.ObservePlaybackCompletion("response-1"); !errors.Is(err, ErrStaleContext) {
		t.Fatalf("old receipt after rotation: %v", err)
	}
	if err := lifecycle.ObservePlaybackCompletion(nextContextID); !errors.Is(err, ErrPlaybackTerminalNotIssued) {
		t.Fatalf("rotation must clear terminal eligibility: %v", err)
	}
	lifecycle.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: nextContextID, Text: "answer"})
	if err := lifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: nextContextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}, func(proto.Message) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.ObservePlaybackCompletion(nextContextID); err != nil {
		t.Fatalf("new response receipt rejected: %v", err)
	}
	if lifecycle.State() != MessageStateAssistantIdle {
		t.Fatalf("receipt changed idle state: %s", lifecycle.State())
	}
}

func TestMessageLifecycle_ConcurrentPlaybackCompletionAcceptsOnce(t *testing.T) {
	lifecycle := NewMessageLifecycleWithContext("response-1", type_enums.AudioMode)
	t.Cleanup(func() { lifecycle.FailAssistantMessage("response-1") })
	lifecycle.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "response-1", Text: "answer"})
	if err := lifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "response-1", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}, func(proto.Message) error { return nil }); err != nil {
		t.Fatal(err)
	}
	var accepted atomic.Int32
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			err := lifecycle.ObservePlaybackCompletion("response-1")
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
	lifecycle := NewMessageLifecycleWithContext("response-1", type_enums.AudioMode)
	t.Cleanup(func() { lifecycle.FailAssistantMessage(lifecycle.ContextID()) })
	lifecycle.FailAssistantMessage("response-1")
	if err := lifecycle.ObservePlaybackCompletion("response-1"); !errors.Is(err, ErrPlaybackTerminalNotIssued) {
		t.Fatalf("revoked terminal must reject receipt: %v", err)
	}
	_, nextContextID, err := lifecycle.RotateContext()
	if err != nil {
		t.Fatal(err)
	}
	lifecycle.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: nextContextID, Text: "answer"})
	if err := lifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: nextContextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}, func(proto.Message) error { return nil }); err != nil {
		t.Fatal(err)
	}
	lifecycle.FailAssistantMessage("response-1")
	if err := lifecycle.ObservePlaybackCompletion(nextContextID); err != nil {
		t.Fatalf("stale revocation affected current terminal: %v", err)
	}
}
