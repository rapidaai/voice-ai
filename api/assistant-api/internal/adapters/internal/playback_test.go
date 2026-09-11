package adapter_internal

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/api/assistant-api/internal/watchdog"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTextToSpeechEndRetainsPlaybackInterruption(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		endAfterPause bool
		confirmSpeech bool
	}{
		{name: "speech after TTS end", confirmSpeech: true},
		{name: "filler after TTS end"},
		{name: "speech with TTS end during pause", endAfterPause: true, confirmSpeech: true},
		{name: "filler with TTS end during pause", endAfterPause: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0, "")
				requestor.messageLifecycle.ConfigureInterruption(true)
				t.Cleanup(func() { requestor.messageLifecycle.CancelInterruption() })
				requestor.endOfSpeechExecutor = &recordingEOSExecutor{}
				streamer := requestor.streamer.(*streamTestStreamer)
				handler := requestorDispatchHandler{r: requestor}
				contextID := requestor.GetID()
				if !scenario.endAfterPause {
					handler.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: contextID})
				}
				handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
					ContextID: contextID, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				if scenario.endAfterPause {
					handler.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: contextID})
				}
				require.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
				require.False(t, requestor.messageLifecycle.CanStartIdleTimeout(contextID))
				if scenario.confirmSpeech {
					handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: contextID, Script: "wait please", Interim: true})
				} else {
					handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: contextID, Script: "um", Interim: true})
					handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
						ContextID: contextID, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
					})
					time.Sleep(adapter_lifecycle.InterruptionDecisionWindow)
				}
				synctest.Wait()
				var controls []proto.Message
				streamer.mu.Lock()
				for _, message := range streamer.sent {
					switch message.(type) {
					case *protos.ConversationPlaybackPause, *protos.ConversationPlaybackContinue, *protos.ConversationPlaybackFlush:
						controls = append(controls, message)
					}
				}
				streamer.mu.Unlock()
				if scenario.confirmSpeech {
					require.Equal(t, []proto.Message{&protos.ConversationPlaybackPause{Id: contextID}, &protos.ConversationPlaybackFlush{Id: contextID}}, controls)
					require.NotEqual(t, contextID, requestor.GetID())
					executor := &toolDispatchTestExecutor{packets: make(chan internal_type.Packet, 1)}
					requestor.assistantExecutor = executor
					handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
						ContextID: requestor.GetID(), Script: "wait please", Interim: false,
					})
					handler.HandleEndOfSpeech(context.Background(), internal_type.EndOfSpeechPacket{
						ContextID: requestor.GetID(), Speech: "wait please",
					})
					require.Equal(t, adapter_lifecycle.MessageStateUserFinished, requestor.messageLifecycle.State())
					for _, packet := range drainIngressPackets(requestor) {
						if input, ok := packet.(internal_type.UserInputPacket); ok {
							handler.HandleUserInput(context.Background(), input)
						}
					}
					synctest.Wait()
					require.Equal(t, adapter_lifecycle.MessageStateAssistantGenerating, requestor.messageLifecycle.State())
					require.Len(t, executor.packets, 1)
					require.Equal(t, internal_type.UserInputPacket{ContextID: requestor.GetID(), Text: "wait please"}, <-executor.packets)
				} else {
					require.Equal(t, []proto.Message{&protos.ConversationPlaybackPause{Id: contextID}, &protos.ConversationPlaybackContinue{Id: contextID}}, controls)
					require.Equal(t, contextID, requestor.GetID())
					require.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
					require.False(t, requestor.messageLifecycle.CanStartIdleTimeout(contextID))
				}
			})
		})
	}
}

func TestTextToSpeechEndLateReceiptDoesNotDismissLaterPlayback(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.assistant = &internal_assistant_entity.Assistant{}
		requestor.assistantConversation = &internal_conversation_entity.AssistantConversation{}
		requestor.observabilityRecorder = &recordingObservabilityRecorder{}
		requestor.messageLifecycle.ConfigureInterruption(true)
		t.Cleanup(func() { requestor.messageLifecycle.CancelInterruption() })
		handler := requestorDispatchHandler{r: requestor}
		contextID := requestor.GetID()
		handler.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: contextID})
		handler.HandleTextToSpeechAudio(context.Background(), internal_type.TextToSpeechAudioPacket{ContextID: contextID, AudioChunk: []byte{0, 0, 0, 0}})
		handler.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: contextID})
		completion := internal_type.PlaybackCompletedPacket{ContextID: contextID, CompletedAt: time.Now(), ReceivedAt: time.Now()}
		handler.HandlePlaybackCompleted(context.Background(), completion)
		handler.HandlePlaybackCompleted(context.Background(), completion)
		require.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
		require.False(t, requestor.messageLifecycle.CanStartIdleTimeout(contextID))
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
			ContextID: contextID, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		})
		streamer := requestor.streamer.(*streamTestStreamer)
		streamer.mu.Lock()
		defer streamer.mu.Unlock()
		require.IsType(t, &protos.ConversationPlaybackPause{}, streamer.sent[len(streamer.sent)-1])
	})
}

func TestTalk_PlaybackCompletionBypassesFullControlQueue(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		timestamp *timestamppb.Timestamp
	}{
		{name: "streamer timestamp", timestamp: timestamppb.New(time.Unix(100, 500))},
		{name: "missing timestamp"},
		{name: "invalid timestamp", timestamp: &timestamppb.Timestamp{Nanos: -1}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			requestor := newInterruptionTestRequestor("")
			requestor.sessionLifecycle = adapter_lifecycle.NewSessionLifecycle()
			requestor.sessionCtx, requestor.cancelSession = context.WithCancel(context.Background())
			t.Cleanup(requestor.cancelSession)
			requestor.assistant = &internal_assistant_entity.Assistant{}
			requestor.assistant.Id = 10
			requestor.assistantConversation = &internal_conversation_entity.AssistantConversation{}
			requestor.assistantConversation.Id = 20
			recorder := &recordingObservabilityRecorder{}
			requestor.observabilityRecorder = recorder
			requestor.messageLifecycle.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "ctx-active", Text: "answer"})
			require.NoError(t, requestor.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "ctx-active", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}, func(proto.Message) error { return nil }))
			t.Cleanup(func() { requestor.messageLifecycle.FailAssistantMessage("ctx-active") })
			requestor.streamer = &streamTestStreamer{recv: []proto.Message{
				&protos.ConversationPlaybackComplete{},
				&protos.ConversationPlaybackComplete{Id: "ctx-stale"},
				&protos.ConversationPlaybackComplete{Id: "ctx-active", Time: scenario.timestamp},
				&protos.ConversationPlaybackComplete{Id: "ctx-active", Time: scenario.timestamp},
			}}
			for range 300 {
				requestor.channels.OnControl(adapter_channel.Envelope{
					Ctx: context.Background(), Pkt: internal_type.SpeechToTextStartPacket{ContextID: "queued"},
				})
			}
			require.Equal(t, 256, requestor.channels.ControlChannel().Len())
			startedAt := time.Now()
			require.NoError(t, requestor.Talk(context.Background(), nil))
			finishedAt := time.Now()

			require.Len(t, recorder.records, 1)
			require.Len(t, recorder.scopes, 1)
			scope, ok := recorder.scopes[0].(observability.MessageScope)
			require.True(t, ok)
			assert.EqualValues(t, 10, scope.AssistantID)
			assert.EqualValues(t, 20, scope.ConversationID)
			assert.Equal(t, "ctx-active", scope.MessageID)
			assert.Equal(t, observability.MessageRoleAssistant, scope.Role)
			event, ok := recorder.records[0].(observability.RecordEvent)
			require.True(t, ok)
			assert.Equal(t, observability.EventName("playback.completed"), event.Event)
			assert.Equal(t, "ctx-active", event.Attributes["message_id"])
			receivedAt, err := time.Parse(time.RFC3339Nano, event.Attributes["received_at"])
			require.NoError(t, err)
			assert.False(t, receivedAt.Before(startedAt))
			assert.False(t, receivedAt.After(finishedAt))
			if scenario.timestamp != nil && scenario.timestamp.IsValid() {
				assert.True(t, event.OccurredAt.Equal(scenario.timestamp.AsTime()))
			} else {
				assert.True(t, event.OccurredAt.Equal(receivedAt))
			}
			assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
			assert.Empty(t, drainEgressPackets(requestor))
			packets := drainControlPackets(requestor)
			require.Len(t, packets, 256)
			for _, packet := range packets {
				assert.IsType(t, internal_type.SpeechToTextStartPacket{}, packet)
			}
		})
	}
}

type terminalReceiptTestStreamer struct {
	streamTestStreamer
	onSend func(proto.Message) error
}

func (s *terminalReceiptTestStreamer) Send(message proto.Message) error {
	if err := s.onSend(message); err != nil {
		return err
	}
	return s.streamTestStreamer.Send(message)
}

func TestPlaybackCompletionRequiresTerminalIssuance(t *testing.T) {
	for _, scenario := range []struct {
		name              string
		receiptDuringSend bool
		sendError         error
	}{
		{name: "early receipt does not consume later receipt"},
		{name: "synchronous receipt during send", receiptDuringSend: true},
		{name: "send failure revokes eligibility", sendError: errors.New("terminal send failed")},
		{name: "diagnostic receipt can precede send failure", receiptDuringSend: true, sendError: errors.New("terminal send failed")},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			requestor := newInterruptionTestRequestor("")
			requestor.assistant = &internal_assistant_entity.Assistant{}
			requestor.assistantConversation = &internal_conversation_entity.AssistantConversation{}
			recorder := &recordingObservabilityRecorder{}
			requestor.observabilityRecorder = recorder
			completion := internal_type.PlaybackCompletedPacket{
				ContextID: "ctx-active", CompletedAt: time.Now(), ReceivedAt: time.Now(),
			}
			requestor.dispatch(context.Background(), completion)
			require.Empty(t, recorder.records, "early receipt must not be recorded")
			requestor.messageLifecycle.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "ctx-active", Text: "answer"})
			require.NoError(t, requestor.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "ctx-active", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}, func(proto.Message) error { return nil }))
			t.Cleanup(func() { requestor.messageLifecycle.FailAssistantMessage("ctx-active") })

			streamer := &terminalReceiptTestStreamer{onSend: func(message proto.Message) error {
				terminal, ok := message.(*protos.ConversationAssistantMessage)
				require.True(t, ok)
				require.True(t, terminal.Completed)
				require.IsType(t, &protos.ConversationAssistantMessage_Audio{}, terminal.Message)
				if scenario.receiptDuringSend {
					requestor.dispatch(context.Background(), completion)
					require.Len(t, recorder.records, 1, "receipt during Send must be eligible")
				}
				return scenario.sendError
			}}
			requestor.streamer = streamer
			requestor.dispatch(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: "ctx-active"})
			assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
			policies := drainControlPackets(requestor)
			require.Len(t, policies, 3, "send failure must retain input-policy fallback")
			for _, packet := range policies {
				policy, ok := packet.(internal_type.DispatchPolicyPacket)
				require.True(t, ok)
				assert.Equal(t, internal_type.DispatchActionPassthrough, policy.Policy.Action)
			}

			if scenario.sendError != nil {
				require.ErrorIs(t, requestor.messageLifecycle.ObservePlaybackCompletion("ctx-active"), adapter_lifecycle.ErrPlaybackTerminalNotIssued)
				requestor.dispatch(context.Background(), completion)
				if scenario.receiptDuringSend {
					require.Len(t, recorder.records, 1, "diagnostic observation is not retracted on send failure")
				} else {
					require.Empty(t, recorder.records, "failed issuance must reject later receipt")
				}
				streamer.onSend = func(proto.Message) error { return nil }
				requestor.dispatch(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: "ctx-active"})
				return
			}
			requestor.dispatch(context.Background(), completion)
			requestor.dispatch(context.Background(), completion)
			require.Len(t, recorder.records, 1, "valid receipt must be recorded exactly once")
			assert.Empty(t, drainEgressPackets(requestor))
		})
	}
}

func TestPlaybackCompletionRejectsDuplicateTerminalBeforeSend(t *testing.T) {
	requestor := newInterruptionTestRequestor("")
	requestor.assistant = &internal_assistant_entity.Assistant{}
	requestor.assistantConversation = &internal_conversation_entity.AssistantConversation{}
	recorder := &recordingObservabilityRecorder{}
	requestor.observabilityRecorder = recorder
	requestor.messageLifecycle.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "ctx-active", Text: "answer"})
	require.NoError(t, requestor.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "ctx-active", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}, func(proto.Message) error { return nil }))
	t.Cleanup(func() { requestor.messageLifecycle.FailAssistantMessage("ctx-active") })
	sendAttempts := 0
	streamer := &terminalReceiptTestStreamer{onSend: func(proto.Message) error {
		sendAttempts++
		if sendAttempts == 2 {
			return errors.New("duplicate terminal send failed")
		}
		return nil
	}}
	requestor.streamer = streamer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	terminal := internal_type.TextToSpeechEndPacket{ContextID: "ctx-active"}
	require.NoError(t, requestor.OnPacket(ctx, terminal, terminal))

	// Exercise the same queued, synchronous dispatch path as the output dispatcher.
	dispatched := 0
	requestor.channels.RunEgress(ctx, func(envelope adapter_channel.Envelope) {
		requestor.dispatch(envelope.Ctx, envelope.Pkt)
		dispatched++
		if dispatched == 2 {
			cancel()
		}
	})
	require.Equal(t, 1, sendAttempts)
	require.Len(t, streamer.sent, 1)
	require.Empty(t, recorder.records)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())

	completion := internal_type.PlaybackCompletedPacket{
		ContextID: "ctx-active", CompletedAt: time.Now(), ReceivedAt: time.Now(),
	}
	requestor.dispatch(context.Background(), completion)
	require.Len(t, recorder.records, 1, "first receipt must remain eligible after a failed duplicate terminal")
	requestor.dispatch(context.Background(), completion)
	require.Len(t, recorder.records, 1, "response dedupe must remain unchanged")
}

func TestPlaybackCompletionPreservesTTSFallbackAndDisabledIdle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor("", 0, "")
		t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
		idleTimeout := uint64(0)
		requestor.assistant.AssistantPhoneDeployment.IdleTimeout = &idleTimeout
		requestor.sessionLifecycle = adapter_lifecycle.NewSessionLifecycle()
		requestor.sessionLifecycle.ConfigureTimeouts(context.Background(), requestor.GetID(),
			&requestor.assistant.AssistantPhoneDeployment.AssistantDeploymentBehavior, requestor.OnPacket)
		requestor.ttsCompletionWatchdog = watchdog.NewTTSCompletionWatchdog(
			watchdog.WithOnPacket(requestor.OnPacket),
			watchdog.WithMinimumTimeout(time.Millisecond),
			watchdog.WithGracePeriod(time.Millisecond),
		)
		t.Cleanup(func() {
			requestor.sessionLifecycle.CloseTimeouts()
			requestor.ttsCompletionWatchdog.Cancel()
		})
		require.True(t, requestor.ttsCompletionWatchdog.Start("ctx-active", time.Second))
		requestor.dispatch(context.Background(), internal_type.PlaybackCompletedPacket{
			ContextID: "ctx-active", CompletedAt: time.Now(), ReceivedAt: time.Now(),
		})
		assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
		assert.Empty(t, drainControlPackets(requestor))
		assert.Empty(t, drainEgressPackets(requestor))

		time.Sleep(2 * time.Second)
		synctest.Wait()
		packets := drainEgressPackets(requestor)
		require.Len(t, packets, 1)
		require.IsType(t, internal_type.TextToSpeechErrorPacket{}, packets[0])
		requestor.dispatch(context.Background(), packets[0])
		assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
		requestor.dispatch(context.Background(), internal_type.StartIdleTimeoutPacket{ContextID: "ctx-active"})
		time.Sleep(time.Minute)
		synctest.Wait()
		for _, packet := range drainEgressPackets(requestor) {
			assert.NotEqual(t, internal_type.PacketNameTextToSpeechEnd, packet.PacketName())
		}
		assert.Zero(t, requestor.sessionLifecycle.IdleTimeoutCount())
		require.Len(t, requestor.streamer.(*streamTestStreamer).sent, 1)
	})
}

func TestTextToSpeechEndSendsTerminalAudioAndRetainsActivePlayback(t *testing.T) {
	requestor := newInterruptionTestRequestor("")
	requestor.messageLifecycle.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "ctx-active", Text: "answer"})
	require.NoError(t, requestor.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{Id: "ctx-active", Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}}}, func(proto.Message) error { return nil }))
	t.Cleanup(func() { requestor.messageLifecycle.FailAssistantMessage("ctx-active") })
	handler := requestorDispatchHandler{r: requestor}
	streamer := requestor.streamer.(*streamTestStreamer)
	handler.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: "ctx-stale"})
	assert.Empty(t, streamer.sent)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())

	handler.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: "ctx-active"})
	require.Len(t, streamer.sent, 1)
	message, ok := streamer.sent[0].(*protos.ConversationAssistantMessage)
	require.True(t, ok)
	assert.Equal(t, "ctx-active", message.Id)
	assert.True(t, message.Completed)
	require.IsType(t, &protos.ConversationAssistantMessage_Audio{}, message.Message)
	assert.Empty(t, message.GetAudio())
	wire, err := proto.Marshal(message)
	require.NoError(t, err)
	decoded := &protos.ConversationAssistantMessage{}
	require.NoError(t, proto.Unmarshal(wire, decoded))
	assert.IsType(t, &protos.ConversationAssistantMessage_Audio{}, decoded.Message)
	assert.True(t, decoded.Completed)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())

	handler.HandlePlaybackCompleted(context.Background(), internal_type.PlaybackCompletedPacket{ContextID: "ctx-active"})
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
	assert.Empty(t, drainEgressPackets(requestor))
	policies := drainControlPackets(requestor)
	require.Len(t, policies, 3)
	for _, packet := range policies {
		policy, ok := packet.(internal_type.DispatchPolicyPacket)
		require.True(t, ok)
		assert.Equal(t, internal_type.DispatchActionPassthrough, policy.Policy.Action)
	}
}

func TestPlaybackCompletionFinishesMessageAndStartsIdleTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor("", 0, "")
		idleTimeout := uint64(1)
		requestor.assistant.AssistantPhoneDeployment.IdleTimeout = &idleTimeout
		requestor.sessionLifecycle = adapter_lifecycle.NewSessionLifecycle()
		requestor.sessionLifecycle.ConfigureTimeouts(context.Background(), requestor.GetID(), &requestor.assistant.AssistantPhoneDeployment.AssistantDeploymentBehavior, requestor.OnPacket)
		requestor.messageLifecycle.ConfigurePlaybackCompletion(func(packets ...internal_type.Packet) error {
			return requestor.OnPacket(context.Background(), packets...)
		})
		t.Cleanup(func() {
			requestor.sessionLifecycle.CloseTimeouts()
			requestor.messageLifecycle.FailAssistantMessage("ctx-active")
			requestor.messageLifecycle.StopUnclearInput()
		})
		handler := requestorDispatchHandler{r: requestor}
		requestor.messageLifecycle.AssistantTextCompleted(internal_type.LLMResponseDonePacket{ContextID: "ctx-active", Text: "answer"})
		handler.HandleTextToSpeechDone(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "ctx-active", Text: "answer"})
		handler.HandleTextToSpeechAudio(context.Background(), internal_type.TextToSpeechAudioPacket{ContextID: "ctx-active", AudioChunk: []byte{0, 0}})
		handler.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: "ctx-active"})
		require.Empty(t, drainEgressPackets(requestor))
		time.Sleep(2 * time.Second)
		synctest.Wait()
		require.Empty(t, drainEgressPackets(requestor), "generation must not start idle timing")
		handler.HandlePlaybackCompleted(context.Background(), internal_type.PlaybackCompletedPacket{ContextID: "ctx-active"})
		require.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, requestor.messageLifecycle.State())
		packets := drainEgressPackets(requestor)
		require.Len(t, packets, 1)
		require.Equal(t, internal_type.StartIdleTimeoutPacket{ContextID: "ctx-active"}, packets[0])
		handler.HandleStartIdleTimeout(context.Background(), packets[0].(internal_type.StartIdleTimeoutPacket))
		handler.HandlePlaybackCompleted(context.Background(), internal_type.PlaybackCompletedPacket{ContextID: "ctx-active"})
		require.Empty(t, drainEgressPackets(requestor), "duplicate receipt must not restart idle")
		time.Sleep(2 * time.Second)
		synctest.Wait()
		expirations := drainEgressPackets(requestor)
		require.Len(t, expirations, 1)
		require.IsType(t, internal_type.IdleTimeoutExpiredPacket{}, expirations[0])
		completionMetrics := 0
		for _, packet := range drainBackgroundPackets(requestor) {
			if metric, ok := packet.(internal_type.ObservabilityMetricRecordPacket); ok {
				for _, value := range metric.Record.Metrics {
					if value.Name == "assistant_turn" {
						completionMetrics++
					}
				}
			}
		}
		require.Equal(t, 1, completionMetrics)
	})
}

func TestStalePlaybackTimeoutDoesNotCloseNewTurn(t *testing.T) {
	requestor := newInterruptionTestRequestor("")
	handler := requestorDispatchHandler{r: requestor}
	handler.HandleError(context.Background(), internal_type.TextToSpeechErrorPacket{ContextID: "previous", Error: errors.New("receipt timed out"), Type: internal_type.TTSPlaybackTimeout})
	assert.Empty(t, requestor.streamer.(*streamTestStreamer).sent)
	assert.Empty(t, drainEgressPackets(requestor))
	assert.Equal(t, "ctx-active", requestor.GetID())
}
