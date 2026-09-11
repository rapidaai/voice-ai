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
	l := NewMessageLifecycleWithContext("active", type_enums.TextMode)
	input, packets := l.AcceptUserInput(internal_type.UserInputPacket{ContextID: "stale", Text: "hello"})
	assert.Empty(t, input.ContextID)
	assert.Empty(t, packets)
	assert.Equal(t, MessageStateAssistantIdle, l.State())
	input, packets = l.AcceptUserInput(internal_type.UserInputPacket{Text: "hello"})
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
	l := NewMessageLifecycleWithContext("active", type_enums.AudioMode)
	assert.Empty(t, l.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "stale", Text: "old"}))
	for _, packet := range []internal_type.Packet{
		internal_type.LLMResponseDonePacket{ContextID: "active", Text: "answer"},
		internal_type.InjectMessagePacket{ContextID: "active", Text: "answer"},
	} {
		l = NewMessageLifecycleWithContext("active", type_enums.AudioMode)
		packets := l.AssistantTextCompleted(packet)
		require.Len(t, packets, 1)
		assert.Equal(t, internal_type.MessageCreatePacket{ContextID: "active", MessageRole: "assistant", Text: "answer"}, packets[0])
		assert.Empty(t, l.AssistantTextCompleted(packet))
	}
}

func TestMessageTurn_PromptAdmissionOwnsContextAndContent(t *testing.T) {
	l := NewMessageLifecycleWithContext("active", type_enums.AudioMode)
	require.NoError(t, l.UserListening("active"))
	_, _, err := l.Prompt(internal_type.UnclearInputExpiredPacket{ContextID: "active"})
	require.ErrorIs(t, err, ErrInvalidTransition)
	assert.Equal(t, "active", l.ContextID())
	promptText := "Please repeat that."
	l.ConfigureUnclearInput(context.Background(), &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputMessage: &promptText}, nil)
	defer l.StopUnclearInput()
	_, _, err = l.Prompt(internal_type.UnclearInputExpiredPacket{ContextID: "stale"})
	require.ErrorIs(t, err, ErrStaleContext)
	turn, prompt, err := l.Prompt(internal_type.UnclearInputExpiredPacket{ContextID: "active"})
	require.NoError(t, err)
	assert.Equal(t, "active", turn.PreviousContextID)
	assert.NotEqual(t, "active", turn.ContextID)
	assert.Equal(t, l.ContextID(), prompt.ContextID)
	assert.Equal(t, promptText, prompt.Text)
	assert.EqualValues(t, 1, l.UserPromptCount())
}
