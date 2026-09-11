package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMessageOptionsDefaults(t *testing.T) {
	message := NewMessageLifecycle()
	assert.NotEmpty(t, message.ContextID())
	assert.NotEqual(t, message.ContextID(), NewMessageLifecycle().ContextID())
	assert.Equal(t, type_enums.TextMode, message.Mode())
	assert.Equal(t, MessageStateAssistantIdle, message.State())
	assert.False(t, message.InterruptionEnabled())

	message = NewMessageLifecycle(nil, WithContextID(""), WithMode(""))
	assert.NotEmpty(t, message.ContextID())
	assert.Equal(t, type_enums.TextMode, message.Mode())

	message = NewMessageLifecycle(
		WithContextID("first"), WithMode(type_enums.TextMode), WithInterruption(false),
		WithContextID("second"), WithMode(type_enums.AudioMode), WithInterruption(true),
	)
	assert.Equal(t, "second", message.ContextID())
	assert.Equal(t, type_enums.AudioMode, message.Mode())
	assert.True(t, message.InterruptionEnabled())
}

func TestMessageOptionsCallbacksWaitForEvents(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		callbackFirst bool
	}{
		{name: "callback first", callbackFirst: true},
		{name: "callback last"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var output []proto.Message
			var packets []internal_type.Packet
			var message MessageLifecycle
			options := []MessageOption{
				WithContextID("message"), WithMode(type_enums.TextMode),
				WithOnPacket(func(emitted ...internal_type.Packet) error {
					assert.Equal(t, "message", message.ContextID())
					packets = append(packets, emitted...)
					return nil
				}),
			}
			sendOption := WithSend(func(packet proto.Message) error {
				assert.Equal(t, "message", message.ContextID())
				output = append(output, packet)
				return nil
			})
			if scenario.callbackFirst {
				options = append([]MessageOption{sendOption}, options...)
			} else {
				options = append(options, sendOption)
			}
			message = NewMessageLifecycle(options...)
			require.Empty(t, output)
			require.Empty(t, packets)
			require.NoError(t, message.OnGenerationStarted("message"))
			message.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "message", Text: "answer"})
			require.NoError(t, message.SendPlaybackControl(&protos.ConversationPlaybackPause{Id: "message"}))
			require.NoError(t, message.SendAssistantMessage(&protos.ConversationAssistantMessage{
				Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "answer"},
			}))
			require.Empty(t, packets, "paused output cannot complete")
			require.NoError(t, message.SendPlaybackControl(&protos.ConversationPlaybackContinue{Id: "message"}))
			require.Len(t, output, 3)
			assert.IsType(t, &protos.ConversationPlaybackPause{}, output[0])
			assert.IsType(t, &protos.ConversationAssistantMessage{}, output[1])
			assert.IsType(t, &protos.ConversationPlaybackContinue{}, output[2])
			require.Len(t, packets, 2)
			assert.Equal(t, internal_type.StartIdleTimeoutPacket{ContextID: "message"}, packets[1])
		})
	}
}

func TestMessageOptionsLoadBehaviorAtInitialization(t *testing.T) {
	timeout := 1.5
	prompt := "Please repeat that."
	loadError := errors.New("deployment unavailable")
	var behaviorError error
	loadCount := 0
	packetCount := 0
	var message MessageLifecycle
	message = NewMessageLifecycle(WithContextID("message"),
		WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
			loadCount++
			return &internal_assistant_entity.AssistantDeploymentBehavior{
				UnclearInputTimeout: &timeout, UnclearInputMessage: &prompt,
			}, behaviorError
		}),
		WithOnPacket(func(packets ...internal_type.Packet) error {
			assert.Equal(t, "message", message.ContextID())
			packetCount += len(packets)
			return nil
		}),
	)
	defer message.Close("message")
	require.Zero(t, loadCount)
	require.Zero(t, packetCount)
	require.NoError(t, message.Initialize(context.Background()))
	assert.Equal(t, 1, loadCount)
	assert.Equal(t, 1, packetCount)
	state := message.(*messageLifecycle)
	assert.Equal(t, 1500*time.Millisecond, state.unclearInputTimeout)
	assert.Equal(t, prompt, state.unclearInputPrompt)
	watchdog := state.unclearInputWatchdog
	require.NotNil(t, watchdog)

	behaviorError = loadError
	require.ErrorIs(t, message.Initialize(context.Background()), loadError)
	assert.Equal(t, 2, loadCount)
	assert.Equal(t, 1, packetCount)
	assert.Same(t, watchdog, state.unclearInputWatchdog)
	assert.Equal(t, prompt, state.unclearInputPrompt)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, message.Initialize(ctx), context.Canceled)
	assert.Equal(t, 2, loadCount)
	assert.Same(t, watchdog, state.unclearInputWatchdog)
}

func TestMessageOptionsMissingSenderDoesNotMutateOutput(t *testing.T) {
	message := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode)).(*messageLifecycle)
	require.NoError(t, message.OnGenerationStarted("message"))
	for _, control := range []proto.Message{
		&protos.ConversationPlaybackPause{Id: "message"},
		&protos.ConversationPlaybackContinue{Id: "message"},
		&protos.ConversationPlaybackFlush{Id: "message"},
	} {
		require.ErrorIs(t, message.SendPlaybackControl(control), ErrSenderNotConfigured)
	}
	require.ErrorIs(t, message.SendAssistantMessage(&protos.ConversationAssistantMessage{
		Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{},
	}), ErrSenderNotConfigured)
	assert.Equal(t, assistantOutputState{started: true}, message.output)
	assert.Equal(t, MessageStateAssistantGenerating, message.State())
}

func TestMessageOptionsMissingDispatcherKeepsPendingTurn(t *testing.T) {
	message := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode), WithInterruption(true))
	require.NoError(t, message.OnGenerationStarted("message"))
	decision := message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "message", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}, "")
	require.NotNil(t, decision.Pause)
	turn, _, _ := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "message", Script: "stop"}, true)
	require.NotNil(t, turn)
	require.ErrorIs(t, message.OnTurnChange(context.Background(), *turn), ErrDispatcherNotConfigured)
	assert.Equal(t, "message", message.ContextID())
	assert.Equal(t, "message", message.CancelInterruption())
}

func TestMessageOptionsDispatchReceivesEventContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var receivedContext context.Context
	var receivedPacket internal_type.Packet
	var message MessageLifecycle
	message = NewMessageLifecycle(WithContextID("message"), WithDispatch(func(ctx context.Context, packet internal_type.Packet) {
		assert.Equal(t, "message", message.ContextID())
		receivedContext = ctx
		receivedPacket = packet
	}))
	turn := internal_type.TurnChangePacket{ContextID: "message", Time: time.Now()}
	require.NoError(t, message.OnTurnChange(ctx, turn))
	assert.Same(t, ctx, receivedContext)
	assert.Equal(t, turn, receivedPacket)
}

func TestMessageOptionsMissingExpiryCallbackDoesNotLeavePauseUnresolved(t *testing.T) {
	message := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode), WithInterruption(true))
	require.NoError(t, message.OnGenerationStarted("message"))
	decision := message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "message", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}, "")
	require.NotNil(t, decision.Pause)
	turn := message.OnPlaybackPaused(*decision.Pause, nil)
	require.NotNil(t, turn)
	assert.True(t, turn.InterruptionDecision)
	assert.Equal(t, decision.Pause.ContextID, turn.PreviousContextID)
	assert.Equal(t, decision.Pause.Sequence, turn.InterruptionSequence)
	assert.Nil(t, message.OnPlaybackPaused(*decision.Pause, nil), "duplicate pause results cannot repeat the decision")
	assert.Equal(t, "message", message.CancelInterruption())
}
