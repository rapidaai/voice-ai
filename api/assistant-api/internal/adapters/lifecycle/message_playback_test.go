package lifecycle

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMessagePlaybackCompletesAfterFinalDelivery(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		duringSend    bool
		sendFails     bool
		paused        bool
		flushed       bool
		authoritative bool
	}{
		{name: "receipt after send", authoritative: true},
		{name: "receipt during send", duringSend: true, authoritative: true},
		{name: "send fails after receipt", duringSend: true, sendFails: true, authoritative: true},
		{name: "paused receipt waits for continue", paused: true, authoritative: true},
		{name: "paused receipt discarded on flush", paused: true, flushed: true, authoritative: true},
		{name: "unverified receipt diagnostic only"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			l := NewMessageLifecycleWithContext("message", type_enums.AudioMode)
			t.Cleanup(func() { l.FailAssistantMessage("message") })
			var packets []internal_type.Packet
			l.ConfigurePlaybackCompletion(scenario.authoritative, func(emitted ...internal_type.Packet) error { packets = append(packets, emitted...); return nil })
			require.NoError(t, l.AssistantGenerating("message"))
			require.ErrorIs(t, l.ObservePlaybackCompletion("message"), ErrPlaybackTerminalNotIssued)
			send := func(proto.Message) error { return nil }
			require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}, send))
			l.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "message", Text: "answer"})
			require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "answer"}}, send))
			require.Empty(t, packets, "text completion cannot complete audio")
			if scenario.paused {
				require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackPause{Id: "message"}, send))
			}
			err := l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{}}, func(proto.Message) error {
				if scenario.duringSend {
					require.NoError(t, l.ObservePlaybackCompletion("message"))
					require.Empty(t, packets, "Send outcome is not yet known")
				}
				if scenario.sendFails {
					return errors.New("write failed")
				}
				return nil
			})
			if scenario.sendFails {
				require.Error(t, err)
				require.ErrorIs(t, l.ObservePlaybackCompletion("message"), ErrPlaybackTerminalNotIssued)
			} else {
				require.NoError(t, err)
				if !scenario.duringSend {
					require.NoError(t, l.ObservePlaybackCompletion("message"))
				}
				require.ErrorIs(t, l.ObservePlaybackCompletion("message"), ErrDuplicatePlaybackCompletion)
			}
			if scenario.paused {
				require.Empty(t, packets)
				if scenario.flushed {
					require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackFlush{Id: "message"}, send))
				} else {
					require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackContinue{Id: "message"}, send))
				}
			}
			if !scenario.sendFails && !scenario.flushed && scenario.authoritative {
				require.Len(t, packets, 2)
				assert.IsType(t, internal_type.ObservabilityMetricRecordPacket{}, packets[0])
				assert.Equal(t, internal_type.StartIdleTimeoutPacket{ContextID: "message"}, packets[1])
				assert.True(t, l.CanStartIdleTimeout("message"))
			} else {
				require.Empty(t, packets)
			}
			require.Error(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{1, 2}}}, send))
		})
	}
}

func TestMessagePlaybackDeadlineExcludesPause(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := NewMessageLifecycleWithContext("message", type_enums.AudioMode)
		var packets []internal_type.Packet
		l.ConfigurePlaybackCompletion(true, func(emitted ...internal_type.Packet) error { packets = append(packets, emitted...); return nil })
		require.NoError(t, l.AssistantGenerating("message"))
		send := func(proto.Message) error { return nil }
		require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}, send))
		l.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "message", Text: "answer"})
		require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "answer"}}, send))
		require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{}}, send))
		time.Sleep(time.Second)
		require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackPause{Id: "message"}, send))
		time.Sleep(time.Minute)
		synctest.Wait()
		require.Empty(t, packets)
		require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackContinue{Id: "message"}, send))
		time.Sleep(5 * time.Second)
		synctest.Wait()
		require.Len(t, packets, 1)
		timeout := packets[0].(internal_type.TextToSpeechErrorPacket)
		assert.Equal(t, internal_type.TTSPlaybackTimeout, timeout.Type)
		assert.False(t, timeout.IsRecoverable())
		assert.False(t, l.CanStartIdleTimeout("message"))
		require.Error(t, l.ObservePlaybackCompletion("message"))
	})
}

func TestMessagePlaybackEmptyAndTextOnly(t *testing.T) {
	for _, mode := range []type_enums.MessageMode{type_enums.AudioMode, type_enums.TextMode} {
		for _, text := range []string{"", "answer"} {
			l := NewMessageLifecycleWithContext("message", mode)
			var packets []internal_type.Packet
			l.ConfigurePlaybackCompletion(true, func(emitted ...internal_type.Packet) error { packets = append(packets, emitted...); return nil })
			require.NoError(t, l.AssistantGenerating("message"))
			l.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "message", Text: text})
			send := func(proto.Message) error { return nil }
			require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: text}}, send))
			if mode.Audio() && text != "" {
				require.Empty(t, packets)
				require.ErrorContains(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "message", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{}}, send), "without audio")
			} else {
				require.Len(t, packets, 2)
			}
		}
	}
}

func TestMessagePlaybackNextSpeechGetsFreshMessageID(t *testing.T) {
	for _, source := range []string{"vad", "stt"} {
		t.Run(source, func(t *testing.T) {
			l := NewMessageLifecycleWithContext("completed", type_enums.AudioMode)
			l.ConfigureInterruption(true)
			l.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "completed"})
			require.NoError(t, l.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "completed", Completed: true, Message: &protos.ConversationAssistantMessage_Text{}}, func(proto.Message) error { return nil }))
			require.True(t, l.CanStartIdleTimeout("completed"))
			if source == "vad" {
				admitted, packets, pause := l.ObserveVAD(internal_type.InterruptionDetectedPacket{ContextID: "completed", Event: internal_type.InterruptionEventStart})
				require.Nil(t, pause)
				require.Equal(t, l.ContextID(), admitted.ContextID)
				require.Len(t, packets, 3)
			} else {
				turn, admitted, _ := l.ObserveSpeech(internal_type.SpeechToTextPacket{ContextID: "completed", Script: "next question"}, true)
				require.True(t, admitted)
				require.NotNil(t, turn)
				require.False(t, turn.InterruptionDecision)
			}
			require.NotEqual(t, "completed", l.ContextID())
			require.ErrorIs(t, l.ObservePlaybackCompletion("completed"), ErrStaleContext)
			require.NoError(t, l.CompleteUserSpeech(internal_type.EndOfSpeechPacket{ContextID: l.ContextID(), Speech: "next question"}))
			_, packets := l.AcceptUserInput(internal_type.UserInputPacket{ContextID: l.ContextID(), Text: "next question"})
			require.NotEmpty(t, packets)
			require.NoError(t, l.AssistantGenerating(l.ContextID()))
		})
	}
}
