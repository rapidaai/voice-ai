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

func TestMessageTurn_SpeechSupersedesQueuedPrompt(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		text     string
		interim  bool
		stale    bool
		generate bool
	}{
		{name: "interim", text: "I am here", interim: true},
		{name: "filler", text: "um", interim: true},
		{name: "final", text: "I am here"},
		{name: "response started", text: "I am here", generate: true},
		{name: "blank", text: " "},
		{name: "stale", text: "old speech", stale: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			message := NewMessageLifecycle(WithContextID("response"), WithMode(type_enums.AudioMode))
			_, prompt, err := message.OnPrompt(internal_type.IdleTimeoutExpiredPacket{ContextID: "response"})
			require.NoError(t, err)
			prompt.Text = "Are you still there?"
			require.Equal(t, MessageStateAssistantPrompted, message.State())
			require.False(t, message.CanStartIdleTimeout(prompt.ContextID))
			transcript := internal_type.SpeechToTextPacket{ContextID: prompt.ContextID, Script: scenario.text, Interim: scenario.interim}
			if scenario.stale {
				transcript.ContextID = "response"
			}
			turn, err := message.OnTranscriptReceived(transcript)
			if scenario.stale || scenario.text == " " {
				if scenario.stale {
					require.ErrorIs(t, err, ErrStaleContext)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, prompt.ContextID, message.ContextID())
				injected, err := message.OnMessageInjected(prompt)
				require.NoError(t, err)
				require.False(t, injected.Interim)
				return
			}
			require.NoError(t, err)
			require.Equal(t, prompt.ContextID, turn.PreviousContextID)
			require.NotEqual(t, prompt.ContextID, turn.ContextID)
			require.Equal(t, MessageStateUserListening, message.State())
			if scenario.generate {
				require.NoError(t, message.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: turn.ContextID, Speech: scenario.text}))
				require.NoError(t, message.OnGenerationStarted(turn.ContextID))
			}
			_, err = message.OnMessageInjected(prompt)
			require.ErrorIs(t, err, ErrStaleContext)
			require.Equal(t, turn.ContextID, message.ContextID())
			if scenario.generate {
				require.Equal(t, MessageStateAssistantGenerating, message.State())
			} else {
				require.Equal(t, MessageStateUserListening, message.State())
			}
		})
	}
}

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
		{name: "accepted filler extends", text: "Um, HMM... uh!", interim: true, deliver: true, extends: true},
		{name: "accepted blank does not stop", text: " \t", deliver: true},
		{name: "accepted final stops", text: "wait please", deliver: true, finishes: true},
		{name: "accepted filler final stops", text: "um", deliver: true, finishes: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				expired := make(chan internal_type.UnclearInputExpiredPacket, 2)
				timeout := 1.0
				l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode),
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
				decision, admitted := l.OnUserSpeech(interim)
				require.NotNil(t, decision)
				require.Empty(t, admitted.ContextID)
				require.True(t, l.beginInterruptedTurn(*decision))
				committed, ok := l.commitInterruptedTurn(*decision)
				require.True(t, ok)
				require.Len(t, l.finishInterruptedTurn(committed), 2)
				require.Empty(t, l.finishInterruptedTurn(committed))
				interim.ContextID = committed.ContextID
				decision, admitted = l.OnUserSpeech(interim)
				assert.Nil(t, decision)
				require.Equal(t, committed.ContextID, admitted.ContextID)
				admittedInterim := admitted
				time.Sleep(2 * time.Second)
				synctest.Wait()
				assert.Empty(t, expired, "admission must not start the unclear timer")
				admittedTurn, err := l.OnTranscriptReceived(admittedInterim)
				require.NoError(t, err)
				assert.Equal(t, committed.ContextID, admittedTurn.ContextID)
				time.Sleep(500 * time.Millisecond)
				packet := internal_type.SpeechToTextPacket{ContextID: admittedTurn.ContextID, Script: scenario.text, Interim: scenario.interim}
				decision, admitted = l.OnUserSpeech(packet)
				assert.Nil(t, decision)
				require.Equal(t, admittedTurn.ContextID, admitted.ContextID)
				if scenario.deliver {
					acceptedTurn, err := l.OnTranscriptReceived(admitted)
					require.NoError(t, err)
					assert.Equal(t, admittedTurn.ContextID, acceptedTurn.ContextID)
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
					assert.Equal(t, admittedTurn.ContextID, (<-expired).ContextID)
				}
			})
		})
	}
}

func TestMessageTurn_AdmittedTranscriptRetainsContextAfterUnclearPrompt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		expired := make(chan internal_type.UnclearInputExpiredPacket, 1)
		timeout, promptText := 1.0, "Please repeat"
		l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode),
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
		})

		require.NotNil(t, decision)
		require.Empty(t, admitted.ContextID)
		require.True(t, l.beginInterruptedTurn(*decision))
		committed, ok := l.commitInterruptedTurn(*decision)
		require.True(t, ok)
		contextB := committed.ContextID
		replay := l.finishInterruptedTurn(committed)
		require.Len(t, replay, 2)
		admittedTurn, err := l.OnTranscriptReceived(replay[1].(internal_type.SpeechToTextPacket))
		require.NoError(t, err)
		require.Equal(t, contextB, admittedTurn.ContextID)
		require.Empty(t, l.finishInterruptedTurn(committed))
		time.Sleep(500 * time.Millisecond)
		final := internal_type.SpeechToTextPacket{Script: "wait please"}
		decision, admitted = l.OnUserSpeech(final)
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
		admittedTurn, err = l.OnTranscriptReceived(admitted)
		assert.ErrorIs(t, err, ErrStaleContext)
		assert.Empty(t, admittedTurn.ContextID)
		assert.Equal(t, turn.ContextID, l.ContextID())
		assert.Equal(t, MessageStateAssistantPrompted, l.State(), "stale delivery must not change the prompt turn")
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
	synctest.Test(t, func(t *testing.T) {
		var behavior *internal_assistant_entity.AssistantDeploymentBehavior
		expired := make(chan internal_type.UnclearInputExpiredPacket, 1)
		l := NewMessageLifecycle(WithContextID("active"), WithMode(type_enums.AudioMode), WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
			return behavior, nil
		}), WithOnPacket(func(packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if packet, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
					expired <- packet
				}
			}
			return nil
		}))
		require.NoError(t, l.Initialize(context.Background()))
		_, err := l.OnTranscriptReceived(internal_type.SpeechToTextPacket{ContextID: "active", Script: "hello"})
		require.NoError(t, err)
		_, _, err = l.OnPrompt(internal_type.UnclearInputExpiredPacket{ContextID: "active"})
		require.ErrorIs(t, err, ErrInvalidTransition)
		assert.Equal(t, "active", l.ContextID())
		promptText := "Please repeat that."
		timeout := 1.0
		behavior = &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputMessage: &promptText, UnclearInputTimeout: &timeout}
		require.NoError(t, l.Initialize(context.Background()))
		defer l.StopUnclearInput()
		_, _, err = l.OnPrompt(internal_type.UnclearInputExpiredPacket{ContextID: "stale"})
		require.ErrorIs(t, err, ErrStaleContext)
		_, err = l.OnTranscriptReceived(internal_type.SpeechToTextPacket{ContextID: "active", Script: "hello", Interim: true})
		require.NoError(t, err)
		time.Sleep(time.Second)
		synctest.Wait()
		turn, prompt, err := l.OnPrompt(<-expired)
		require.NoError(t, err)
		assert.Equal(t, "active", turn.PreviousContextID)
		assert.NotEqual(t, "active", turn.ContextID)
		assert.Equal(t, l.ContextID(), prompt.ContextID)
		assert.Equal(t, promptText, prompt.Text)
		_, _, err = l.OnPrompt(internal_type.UnclearInputExpiredPacket{ContextID: "active"})
		assert.ErrorIs(t, err, ErrStaleContext)
		assert.Equal(t, turn.ContextID, l.ContextID())
	})
}

func TestMessageTurn_OrdinarySpeechOwnsUnclearCountdown(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		text    string
		interim bool
		stale   bool
		extends bool
		stops   bool
	}{
		{name: "interim restarts", text: "I need to change", interim: true, extends: true},
		{name: "filler interim restarts", text: "um", interim: true, extends: true},
		{name: "blank interim is ignored", text: " ", interim: true},
		{name: "blank final is ignored", text: " "},
		{name: "final cancels", text: "I need to change", stops: true},
		{name: "stale interim is ignored", text: "old words", interim: true, stale: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				timeout, promptText := 1.0, "Please repeat that."
				expired := make(chan internal_type.UnclearInputExpiredPacket, 2)
				resets := make(chan internal_type.StopIdleTimeoutPacket, 3)
				message := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode),
					WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
						return &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputTimeout: &timeout, UnclearInputMessage: &promptText}, nil
					}), WithOnPacket(func(packets ...internal_type.Packet) error {
						for _, packet := range packets {
							switch packet := packet.(type) {
							case internal_type.UnclearInputExpiredPacket:
								expired <- packet
							case internal_type.StopIdleTimeoutPacket:
								resets <- packet
							}
						}
						return nil
					}))
				defer message.StopUnclearInput()
				require.NoError(t, message.Initialize(t.Context()))
				message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{ContextID: "message", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart}, "")
				message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{ContextID: "message", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd}, "")
				time.Sleep(2 * time.Second)
				synctest.Wait()
				require.Empty(t, expired, "VAD without a transcript must not create an unclear prompt")
				require.Empty(t, resets, "VAD alone must not reset idle retries")
				require.True(t, message.CanStartIdleTimeout("message"))
				_, err := message.OnTranscriptReceived(internal_type.SpeechToTextPacket{ContextID: "message", Script: "I need", Interim: true})
				require.NoError(t, err)
				require.Equal(t, internal_type.StopIdleTimeoutPacket{ContextID: "message", ResetCount: true}, <-resets)
				time.Sleep(500 * time.Millisecond)
				packet := internal_type.SpeechToTextPacket{ContextID: "message", Script: scenario.text, Interim: scenario.interim}
				if scenario.stale {
					packet.ContextID = "old"
				}
				_, err = message.OnTranscriptReceived(packet)
				if scenario.stale {
					require.ErrorIs(t, err, ErrStaleContext)
				} else {
					require.NoError(t, err)
				}
				message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{ContextID: "message", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd}, "")
				time.Sleep(500 * time.Millisecond)
				synctest.Wait()
				if scenario.extends || scenario.stops {
					require.Empty(t, expired)
					require.Equal(t, internal_type.StopIdleTimeoutPacket{ContextID: "message", ResetCount: true}, <-resets)
				} else {
					require.Len(t, expired, 1)
					require.Empty(t, resets)
				}
				time.Sleep(500 * time.Millisecond)
				synctest.Wait()
				if scenario.stops {
					require.Empty(t, expired, "VAD end after a final must not restart the unclear timer")
					return
				}
				require.Len(t, expired, 1)
				_, prompt, err := message.OnPrompt(<-expired)
				require.NoError(t, err)
				require.Equal(t, promptText, prompt.Text)
			})
		})
	}
}

func TestMessageTurn_FinalInvalidatesQueuedUnclearExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		timeout, promptText := 1.0, "Please repeat."
		expired := make(chan internal_type.UnclearInputExpiredPacket, 1)
		message := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode),
			WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
				return &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputTimeout: &timeout, UnclearInputMessage: &promptText}, nil
			}), WithOnPacket(func(packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
						expired <- packet
					}
				}
				return nil
			}))
		defer message.StopUnclearInput()
		require.NoError(t, message.Initialize(t.Context()))
		_, err := message.OnTranscriptReceived(internal_type.SpeechToTextPacket{ContextID: "message", Script: "hello", Interim: true})
		require.NoError(t, err)
		time.Sleep(time.Second)
		synctest.Wait()
		require.Len(t, expired, 1)
		_, err = message.OnTranscriptReceived(internal_type.SpeechToTextPacket{ContextID: "message", Script: "hello"})
		require.NoError(t, err)
		_, _, err = message.OnPrompt(<-expired)
		require.ErrorIs(t, err, ErrInvalidTransition)
		require.Equal(t, "message", message.ContextID())
	})
}

func TestMessageTurn_CompletedPlaybackPreservesPendingTranscriptContext(t *testing.T) {
	message := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode)).(*messageLifecycle)
	message.output.playback = playbackCompleted
	decision := message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "message", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}, "")
	require.Nil(t, decision.Pause)
	require.Equal(t, "message", message.ContextID())
	require.True(t, message.CanStartIdleTimeout("message"))
	interruption, interim := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "message", Script: "hello", Interim: true})
	require.NotNil(t, interruption)
	turn, err := message.OnTranscriptReceived(interim)
	require.NoError(t, err)
	require.NotEqual(t, "message", turn.ContextID)
	decision = message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "message", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}, "")
	require.NotNil(t, decision.EndOfSpeech)
	require.Equal(t, turn.ContextID, decision.EndOfSpeech.ContextID)
	_, final := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "message", Script: "hello there"})
	require.Equal(t, turn.ContextID, final.ContextID)
	_, err = message.OnTranscriptReceived(final)
	require.NoError(t, err)
	_, stale := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "message", Script: "old result"})
	require.Empty(t, stale.ContextID)
}

func TestMessageTurn_AdmittedSpeechSupersedesPendingUserInput(t *testing.T) {
	message := NewMessageLifecycle(WithContextID("first"), WithMode(type_enums.AudioMode))
	require.NoError(t, message.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: "first", Speech: "wait"}))
	turn, speech := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "first", Script: "and one more thing"})
	require.NotNil(t, turn)
	require.NotEqual(t, "first", speech.ContextID)
	input, packets := message.OnUserInput(internal_type.UserInputPacket{ContextID: "first", Text: "wait"})
	require.Empty(t, input.ContextID)
	require.Empty(t, packets)
	admittedTurn, err := message.OnTranscriptReceived(speech)
	require.NoError(t, err)
	require.Equal(t, speech.ContextID, admittedTurn.ContextID)
	require.Empty(t, admittedTurn.PreviousContextID)
	require.Equal(t, MessageStateUserListening, message.State())
}
