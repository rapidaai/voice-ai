package lifecycle

import (
	"errors"
	"testing"

	type_enums "github.com/rapidaai/pkg/types/enums"
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
