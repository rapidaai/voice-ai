package lifecycle

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageTurn_TranscriptDeliveryOwnsUnclearTimer(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		text     string
		interim  bool
		deliver  bool
		extends  bool
		finishes bool
	}{
		{name: "interim admission does not extend", text: "wait please", interim: true},
		{name: "final admission does not stop", text: "wait please"},
		{name: "accepted interim extends", text: "wait please", interim: true, deliver: true, extends: true},
		{name: "accepted filler does not extend", text: "Um, HMM... uh!", interim: true, deliver: true},
		{name: "accepted blank does not stop", text: " \t", deliver: true},
		{name: "accepted final stops", text: "wait please", deliver: true, finishes: true},
		{name: "accepted filler final stops", text: "um", deliver: true, finishes: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				expired := make(chan internal_type.UnclearInputExpiredPacket, 2)
				timeout := 1.0
				l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
					WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
						return &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputTimeout: &timeout}, nil
					}),
					WithOnPacket(func(packets ...internal_type.Packet) error {
						for _, packet := range packets {
							if packet, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
								expired <- packet
							}
						}
						return nil
					})).(*messageLifecycle)
				t.Cleanup(func() { l.CancelInterruption() })
				require.NoError(t, l.Initialize(t.Context()))
				require.NoError(t, l.OnGenerationStarted("assistant"))
				_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				require.NotNil(t, pause)
				interim := internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait", Interim: true}
				decision, admitted := l.OnUserSpeech(interim, true)
				require.NotNil(t, decision)
				require.Empty(t, admitted.ContextID)
				require.True(t, l.beginInterruptedTurn(*decision))
				committed, ok := l.commitInterruptedTurn(*decision)
				require.True(t, ok)
				require.Len(t, l.finishInterruptedTurn(committed), 2)
				require.Empty(t, l.finishInterruptedTurn(committed))
				interim.ContextID = committed.ContextID
				decision, admitted = l.OnUserSpeech(interim, true)
				assert.Nil(t, decision)
				require.Equal(t, committed.ContextID, admitted.ContextID)
				admittedInterim := admitted
				time.Sleep(2 * time.Second)
				synctest.Wait()
				assert.Empty(t, expired, "admission must not start the unclear timer")
				filler := internal_type.SpeechToTextPacket{ContextID: committed.ContextID, Script: "Um, HMM... uh!", Interim: true}
				decision, admitted = l.OnUserSpeech(filler, true)
				assert.Nil(t, decision)
				require.Equal(t, committed.ContextID, admitted.ContextID)
				contextID, err := l.OnTranscriptReceived(admitted)
				require.NoError(t, err)
				assert.Equal(t, committed.ContextID, contextID)
				time.Sleep(2 * time.Second)
				synctest.Wait()
				assert.Empty(t, expired, "accepted interim filler must not start the unclear timer")
				contextID, err = l.OnTranscriptReceived(admittedInterim)
				require.NoError(t, err)
				assert.Equal(t, committed.ContextID, contextID)
				time.Sleep(500 * time.Millisecond)
				packet := internal_type.SpeechToTextPacket{ContextID: contextID, Script: scenario.text, Interim: scenario.interim}
				decision, admitted = l.OnUserSpeech(packet, true)
				assert.Nil(t, decision)
				require.Equal(t, contextID, admitted.ContextID)
				if scenario.deliver {
					acceptedContext, err := l.OnTranscriptReceived(admitted)
					require.NoError(t, err)
					assert.Equal(t, contextID, acceptedContext)
				}
				time.Sleep(500 * time.Millisecond)
				synctest.Wait()
				if scenario.extends || scenario.finishes {
					assert.Empty(t, expired)
				} else {
					require.Len(t, expired, 1, "the original accepted transcript deadline must remain unchanged")
				}
				time.Sleep(500 * time.Millisecond)
				synctest.Wait()
				if scenario.finishes {
					assert.Empty(t, expired)
				} else {
					require.Len(t, expired, 1)
					assert.Equal(t, contextID, (<-expired).ContextID)
				}
			})
		})
	}
}

func TestMessageTurn_AdmittedTranscriptRetainsContextAfterUnclearPrompt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		expired := make(chan internal_type.UnclearInputExpiredPacket, 1)
		timeout, promptText := 1.0, "Please repeat"
		l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
			WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
				return &internal_assistant_entity.AssistantDeploymentBehavior{
					UnclearInputTimeout: &timeout, UnclearInputMessage: &promptText,
				}, nil
			}),
			WithOnPacket(func(packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
						expired <- packet
					}
				}
				return nil
			})).(*messageLifecycle)
		t.Cleanup(func() { l.CancelInterruption() })
		require.NoError(t, l.Initialize(t.Context()))
		require.NoError(t, l.OnGenerationStarted("assistant"))
		_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
			ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		})
		require.NotNil(t, pause)
		decision, admitted := l.OnUserSpeech(internal_type.SpeechToTextPacket{
			ContextID: "assistant", Script: "wait", Interim: true,
		}, true)
		require.NotNil(t, decision)
		require.Empty(t, admitted.ContextID)
		require.True(t, l.beginInterruptedTurn(*decision))
		committed, ok := l.commitInterruptedTurn(*decision)
		require.True(t, ok)
		contextB := committed.ContextID
		replay := l.finishInterruptedTurn(committed)
		require.Len(t, replay, 2)
		contextID, err := l.OnTranscriptReceived(replay[1].(internal_type.SpeechToTextPacket))
		require.NoError(t, err)
		require.Equal(t, contextB, contextID)
		require.Empty(t, l.finishInterruptedTurn(committed))
		time.Sleep(500 * time.Millisecond)
		final := internal_type.SpeechToTextPacket{Script: "wait please"}
		decision, admitted = l.OnUserSpeech(final, true)
		assert.Nil(t, decision)
		require.Equal(t, contextB, admitted.ContextID)
		final.ContextID = contextB
		assert.Equal(t, final, admitted, "admission must preserve the transcript while capturing its context")
		time.Sleep(500 * time.Millisecond)
		synctest.Wait()
		require.Len(t, expired, 1, "admission alone must not cancel the accepted interim's timer")
		turn, prompt, err := l.OnPrompt(<-expired)
		require.NoError(t, err)
		require.Equal(t, contextB, turn.PreviousContextID)
		require.NotEqual(t, contextB, turn.ContextID)
		assert.Equal(t, turn.ContextID, prompt.ContextID)
		assert.Equal(t, promptText, prompt.Text)
		assert.Equal(t, contextB, admitted.ContextID)
		contextID, err = l.OnTranscriptReceived(admitted)
		assert.ErrorIs(t, err, ErrStaleContext)
		assert.Empty(t, contextID)
		assert.Equal(t, turn.ContextID, l.ContextID())
		assert.Equal(t, MessageStateAssistantIdle, l.State(), "stale delivery must not change the prompt turn")
	})
}

func TestMessageTurn_AcceptedUserInputOwnsCompletionPackets(t *testing.T) {
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
	_, err := l.OnTranscriptReceived(internal_type.SpeechToTextPacket{ContextID: "active", Script: "hello"})
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
