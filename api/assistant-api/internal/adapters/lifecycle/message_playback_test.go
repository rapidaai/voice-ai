package lifecycle

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMessagePlaybackReceiptDoesNotExpireDuringReplay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		received, release := make(chan struct{}), make(chan struct{})
		defer close(release)
		finished := make(chan error, 1)
		timeouts := make(chan internal_type.TextToSpeechErrorPacket, 2)
		idle := make(chan internal_type.StartIdleTimeoutPacket, 2)
		var message MessageLifecycle
		message = NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
			WithSend(func(proto.Message) error { return nil }),
			WithOnPacket(func(packets ...internal_type.Packet) error {
				for _, packet := range packets {
					switch packet := packet.(type) {
					case internal_type.TextToSpeechErrorPacket:
						timeouts <- packet
					case internal_type.StartIdleTimeoutPacket:
						idle <- packet
					}
				}
				return nil
			}),
			WithDispatch(func(_ context.Context, packet internal_type.Packet) {
				switch packet := packet.(type) {
				case internal_type.SpeechToTextPacket:
					contextID, err := message.OnTranscriptReceived(packet)
					assert.NoError(t, err)
					assert.Equal(t, packet.ContextID, contextID)
				case internal_type.UserInputPacket:
					input, _ := message.OnUserInput(packet)
					assert.NotEmpty(t, input.ContextID)
					assert.NoError(t, message.OnGenerationStarted(input.ContextID))
					message.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: input.ContextID, Text: "answer"})
					assert.NoError(t, message.SendAssistantMessage(&protos.ConversationAssistantMessage{
						Id: input.ContextID, Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "answer"},
					}))
					assert.NoError(t, message.SendAssistantMessage(&protos.ConversationAssistantMessage{
						Id: input.ContextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
					}))
					assert.NoError(t, message.OnPlaybackCompleted(input.ContextID))
					close(received)
					<-release
				}
			}))
		defer func() { message.Close(message.ContextID()) }()
		require.NoError(t, message.OnGenerationStarted("assistant"))
		message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
			ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		}, "")
		turn, _ := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait"}, true)
		require.NotNil(t, turn)
		require.True(t, message.HoldInput(internal_type.UserInputPacket{ContextID: "assistant", Text: "wait"}))
		go func() { finished <- message.OnTurnChange(t.Context(), *turn) }()
		<-received
		time.Sleep(6 * time.Second)
		synctest.Wait()
		assert.Empty(t, timeouts)
		assert.Empty(t, idle, "completion must wait until replay releases input")
		release <- struct{}{}
		require.NoError(t, <-finished)
		require.Len(t, idle, 1)
		assert.Equal(t, message.ContextID(), (<-idle).ContextID)
		assert.True(t, message.CanStartIdleTimeout(message.ContextID()))
	})
}

func TestMessagePlaybackCompletesAfterFinalDelivery(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		duringSend bool
		sendFails  bool
		paused     bool
		flushed    bool
	}{
		{name: "receipt after send"},
		{name: "receipt during send", duringSend: true},
		{name: "send fails after receipt", duringSend: true, sendFails: true},
		{name: "paused receipt waits for continue", paused: true},
		{name: "paused receipt discarded on flush", paused: true, flushed: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var packets []internal_type.Packet
			var l MessageLifecycle
			l = NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode),
				WithOnPacket(func(emitted ...internal_type.Packet) error { packets = append(packets, emitted...); return nil }),
				WithSend(func(packet proto.Message) error {
					message, ok := packet.(*protos.ConversationAssistantMessage)
					if !ok || !message.Completed {
						return nil
					}
					if _, audio := message.Message.(*protos.ConversationAssistantMessage_Audio); !audio {
						return nil
					}
					if scenario.duringSend {
						require.NoError(t, l.OnPlaybackCompleted("message"))
						require.Empty(t, packets, "Send outcome is not yet known")
					}
					if scenario.sendFails {
						return errors.New("write failed")
					}
					return nil
				}))
			t.Cleanup(func() { l.OnMessageFailed("message") })
			require.NoError(t, l.OnGenerationStarted("message"))
			require.ErrorIs(t, l.OnPlaybackCompleted("message"), ErrPlaybackTerminalNotIssued)
			require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}))
			l.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "message", Text: "answer"})
			require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "answer"}}))
			require.Empty(t, packets, "text completion cannot complete audio")
			if scenario.paused {
				require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackPause{Id: "message"}))
			}
			err := l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{}})
			if scenario.sendFails {
				require.Error(t, err)
				require.ErrorIs(t, l.OnPlaybackCompleted("message"), ErrPlaybackTerminalNotIssued)
			} else {
				require.NoError(t, err)
				if !scenario.duringSend {
					require.NoError(t, l.OnPlaybackCompleted("message"))
				}
				require.ErrorIs(t, l.OnPlaybackCompleted("message"), ErrDuplicatePlaybackCompletion)
			}
			if scenario.paused {
				require.Empty(t, packets)
				if scenario.flushed {
					require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackFlush{Id: "message"}))
				} else {
					require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackContinue{Id: "message"}))
				}
			}
			if !scenario.sendFails && !scenario.flushed {
				require.Len(t, packets, 2)
				assert.IsType(t, internal_type.ObservabilityMetricRecordPacket{}, packets[0])
				assert.Equal(t, internal_type.StartIdleTimeoutPacket{ContextID: "message"}, packets[1])
				assert.True(t, l.CanStartIdleTimeout("message"))
			} else {
				require.Empty(t, packets)
			}
			require.Error(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1, 2}}}))
		})
	}
}

func TestMessagePlaybackDoesNotRequireConfiguration(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		receipt bool
	}{
		{name: "receipt completes the message", receipt: true},
		{name: "missing receipt fails the message"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				l := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode), WithSend(func(proto.Message) error { return nil }))
				defer l.OnMessageFailed("message")
				require.NoError(t, l.OnGenerationStarted("message"))
				require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{
					Id: "message", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
				}))
				l.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "message", Text: "answer"})
				require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{
					Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "answer"},
				}))
				require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{
					Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{},
				}))
				require.False(t, l.CanStartIdleTimeout("message"))
				if scenario.receipt {
					require.NoError(t, l.OnPlaybackCompleted("message"))
					assert.Equal(t, MessageStateAssistantIdle, l.State())
					assert.True(t, l.CanStartIdleTimeout("message"))
					return
				}
				time.Sleep(6 * time.Second)
				synctest.Wait()
				assert.False(t, l.CanStartIdleTimeout("message"))
				require.ErrorIs(t, l.OnPlaybackCompleted("message"), ErrPlaybackTerminalNotIssued)
				require.ErrorIs(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{
					Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{},
				}), ErrInvalidTransition)
			})
		})
	}
}

func TestMessagePlaybackDeadlineExcludesPause(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var packets []internal_type.Packet
		l := NewMessageLifecycle(WithContextID("message"), WithMode(type_enums.AudioMode),
			WithSend(func(proto.Message) error { return nil }),
			WithOnPacket(func(emitted ...internal_type.Packet) error { packets = append(packets, emitted...); return nil }))
		require.NoError(t, l.OnGenerationStarted("message"))
		require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}))
		l.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "message", Text: "answer"})
		require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "answer"}}))
		require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{}}))
		time.Sleep(time.Second)
		require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackPause{Id: "message"}))
		time.Sleep(time.Minute)
		synctest.Wait()
		require.Empty(t, packets)
		require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackContinue{Id: "message"}))
		time.Sleep(5 * time.Second)
		synctest.Wait()
		require.Len(t, packets, 1)
		timeout := packets[0].(internal_type.TextToSpeechErrorPacket)
		assert.Equal(t, internal_type.TTSPlaybackTimeout, timeout.Type)
		assert.False(t, timeout.IsRecoverable())
		assert.False(t, l.CanStartIdleTimeout("message"))
		require.Error(t, l.OnPlaybackCompleted("message"))
	})
}

func TestMessagePlaybackEmptyAndTextOnly(t *testing.T) {
	for _, mode := range []type_enums.MessageMode{type_enums.AudioMode, type_enums.TextMode} {
		for _, text := range []string{"", "answer"} {
			var packets []internal_type.Packet
			l := NewMessageLifecycle(WithContextID("message"), WithMode(mode),
				WithSend(func(proto.Message) error { return nil }),
				WithOnPacket(func(emitted ...internal_type.Packet) error { packets = append(packets, emitted...); return nil }))
			require.NoError(t, l.OnGenerationStarted("message"))
			l.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "message", Text: text})
			require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: text}}))
			if mode.Audio() && text != "" {
				require.Empty(t, packets)
				require.ErrorContains(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{}}), "without audio")
			} else {
				require.Len(t, packets, 2)
			}
		}
	}
}

func TestMessagePlaybackNextSpeechGetsFreshMessageID(t *testing.T) {
	for _, source := range []string{"vad", "stt"} {
		t.Run(source, func(t *testing.T) {
			l := NewMessageLifecycle(WithContextID("completed"), WithMode(type_enums.AudioMode), WithInterruption(true), WithSend(func(proto.Message) error { return nil }))
			l.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: "completed"})
			require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "completed", Completed: true, Message: &protos.ConversationAssistantMessage_Text{}}))
			require.True(t, l.CanStartIdleTimeout("completed"))
			if source == "vad" {
				decision := l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
					ContextID: "completed", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				}, internal_options.BargeInTriggerVAD)
				require.Nil(t, decision.Pause)
				require.NotNil(t, decision.EndOfSpeech)
				require.Equal(t, l.ContextID(), decision.EndOfSpeech.ContextID)
				require.Len(t, decision.Packets, 3)
			} else {
				turn, admitted := l.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "completed", Script: "next question"}, true)
				require.NotEmpty(t, admitted.ContextID)
				require.NotNil(t, turn)
				assert.Equal(t, turn.ContextID, admitted.ContextID)
				require.False(t, turn.InterruptionDecision)
			}
			require.NotEqual(t, "completed", l.ContextID())
			require.ErrorIs(t, l.OnPlaybackCompleted("completed"), ErrStaleContext)
			require.NoError(t, l.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: l.ContextID(), Speech: "next question"}))
			_, packets := l.OnUserInput(internal_type.UserInputPacket{ContextID: l.ContextID(), Text: "next question"})
			require.NotEmpty(t, packets)
			require.NoError(t, l.OnGenerationStarted(l.ContextID()))
		})
	}
}
