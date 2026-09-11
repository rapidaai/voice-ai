package lifecycle

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageTurn_UserInputAdmissionOwnsCompletionPackets(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("active"), WithMode(type_enums.TextMode))
	input, packets := l.OnUserInput(internal_type.UserInputPacket{ContextID: "stale", Text: "hello"})
	assert.Empty(t, input.ContextID)
	assert.Empty(t, packets)
	assert.Equal(t, MessageStateAssistantIdle, l.State())
	input, packets = l.OnUserInput(internal_type.UserInputPacket{Text: "hello"})
	assert.Equal(t, "active", input.ContextID)
	assert.Equal(t, MessageStateUserFinished, l.State())
	require.Len(t, packets, 4)
	assert.Equal(t, internal_type.StopIdleTimeoutPacket{ContextID: "active", ResetCount: true}, packets[0])
	assert.Equal(t, internal_type.MessageCreatePacket{ContextID: "active", MessageRole: "user", Text: "hello"}, packets[1])
	metric, ok := packets[3].(internal_type.ObservabilityMetricRecordPacket)
	require.True(t, ok)
	assert.Equal(t, "active", metric.ContextID)
}

func TestMessageTurn_AssistantCompletionPacketsRejectStaleContext(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("active"), WithMode(type_enums.AudioMode))
	assert.Empty(t, l.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "stale", Text: "old"}))
	for _, packet := range []internal_type.Packet{
		internal_type.LLMResponseDonePacket{ContextID: "active", Text: "answer"},
		internal_type.InjectMessagePacket{ContextID: "active", Text: "answer"},
	} {
		l = NewMessageLifecycle(WithContextID("active"), WithMode(type_enums.AudioMode))
		packets := l.OnGenerationCompleted(packet)
		require.Len(t, packets, 1)
		assert.Equal(t, internal_type.MessageCreatePacket{ContextID: "active", MessageRole: "assistant", Text: "answer"}, packets[0])
		assert.Empty(t, l.OnGenerationCompleted(packet))
	}
}

func TestMessageTurn_PromptAdmissionOwnsContextAndContent(t *testing.T) {
	var behavior *internal_assistant_entity.AssistantDeploymentBehavior
	l := NewMessageLifecycle(WithContextID("active"), WithMode(type_enums.AudioMode), WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
		return behavior, nil
	}))
	require.NoError(t, l.Initialize(context.Background()))
	_, err := l.OnTranscriptReceived("active", "hello")
	require.NoError(t, err)
	_, _, err = l.OnPrompt(internal_type.UnclearInputExpiredPacket{ContextID: "active"})
	require.ErrorIs(t, err, ErrInvalidTransition)
	assert.Equal(t, "active", l.ContextID())
	promptText := "Please repeat that."
	behavior = &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputMessage: &promptText}
	require.NoError(t, l.Initialize(context.Background()))
	defer l.StopUnclearInput()
	_, _, err = l.OnPrompt(internal_type.UnclearInputExpiredPacket{ContextID: "stale"})
	require.ErrorIs(t, err, ErrStaleContext)
	turn, prompt, err := l.OnPrompt(internal_type.UnclearInputExpiredPacket{ContextID: "active"})
	require.NoError(t, err)
	assert.Equal(t, "active", turn.PreviousContextID)
	assert.NotEqual(t, "active", turn.ContextID)
	assert.Equal(t, l.ContextID(), prompt.ContextID)
	assert.Equal(t, promptText, prompt.Text)
	_, _, err = l.OnPrompt(internal_type.UnclearInputExpiredPacket{ContextID: "active"})
	assert.ErrorIs(t, err, ErrStaleContext)
	assert.Equal(t, turn.ContextID, l.ContextID())
}
