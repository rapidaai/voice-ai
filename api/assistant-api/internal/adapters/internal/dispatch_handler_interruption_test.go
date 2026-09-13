package adapter_internal

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	adapter_router "github.com/rapidaai/api/assistant-api/internal/adapters/router"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestDispatchInterruptionUnclearInputExtendsAndIgnoresEmptyFinal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat", adapter_lifecycle.WithInterruption(true))
		t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
		requestor.endOfSpeechExecutor = &recordingEOSExecutor{}
		streamer := requestor.streamer.(*streamTestStreamer)
		handler := requestorDispatchHandler{r: requestor}
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{Script: "wait", Interim: true})
		synctest.Wait()
		contextID := requestor.GetID()
		time.Sleep(600 * time.Millisecond)
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: contextID, Script: "wait please", Interim: true})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: contextID, Script: "  ", Interim: false})
		synctest.Wait()
		time.Sleep(600 * time.Millisecond)
		synctest.Wait()
		for _, packet := range drainEgressPackets(requestor) {
			_, expired := packet.(internal_type.UnclearInputExpiredPacket)
			assert.False(t, expired)
		}
		time.Sleep(400 * time.Millisecond)
		synctest.Wait()
		var expired internal_type.UnclearInputExpiredPacket
		for _, packet := range drainEgressPackets(requestor) {
			if value, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				expired = value
			}
		}
		require.Equal(t, contextID, expired.ContextID)
		handler.HandleUnclearInputExpired(context.Background(), expired)
		synctest.Wait()
		assert.NotEqual(t, contextID, requestor.GetID())
		var injected internal_type.InjectMessagePacket
		for _, packet := range drainEgressPackets(requestor) {
			if value, ok := packet.(internal_type.InjectMessagePacket); ok {
				injected = value
			}
		}
		assert.Equal(t, "Please repeat", injected.Text)
		assert.Equal(t, requestor.GetID(), injected.ContextID)
		streamer.mu.Lock()
		require.GreaterOrEqual(t, len(streamer.sent), 2)
		assert.Equal(t, protos.ConversationPlaybackControl_PAUSE, streamer.sent[0].(*protos.ConversationPlaybackControl).GetKind())
		assert.Equal(t, protos.ConversationPlaybackControl_FLUSH, streamer.sent[1].(*protos.ConversationPlaybackControl).GetKind())
		streamer.mu.Unlock()
	})
}

func TestDispatchInterruptionCompletedInputStopsUnclearTracking(t *testing.T) {
	for _, completion := range []string{"final", "eos", "input", "text"} {
		t.Run(completion, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat", adapter_lifecycle.WithInterruption(true))
				t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
				requestor.endOfSpeechExecutor = &recordingEOSExecutor{}
				handler := requestorDispatchHandler{r: requestor}
				handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
				handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{Script: "wait", Interim: true})
				synctest.Wait()
				contextID := requestor.GetID()
				switch completion {
				case "final":
					handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: contextID, Script: "Wait please"})
				case "eos":
					handler.HandleEndOfSpeech(context.Background(), internal_type.EndOfSpeechPacket{ContextID: contextID, Speech: "Wait please"})
				case "input":
					handler.HandleUserInput(context.Background(), internal_type.UserInputPacket{ContextID: contextID, Text: "Wait please"})
				case "text":
					handler.HandleUserText(context.Background(), internal_type.UserTextReceivedPacket{ContextID: contextID, Text: "Wait please"})
				}
				synctest.Wait()
				time.Sleep(2 * time.Second)
				synctest.Wait()
				for _, packet := range drainEgressPackets(requestor) {
					_, expired := packet.(internal_type.UnclearInputExpiredPacket)
					assert.False(t, expired)
				}
			})
		})
	}
}

func TestDispatchInterruptionFinalRejectsQueuedExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat", adapter_lifecycle.WithInterruption(true))
		t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
		requestor.endOfSpeechExecutor = &recordingEOSExecutor{}
		handler := requestorDispatchHandler{r: requestor}
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{Script: "wait", Interim: true})
		synctest.Wait()
		contextID := requestor.GetID()
		time.Sleep(time.Second)
		synctest.Wait()
		var expired internal_type.UnclearInputExpiredPacket
		for _, packet := range drainEgressPackets(requestor) {
			if value, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				expired = value
			}
		}
		require.Equal(t, contextID, expired.ContextID)
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: contextID, Script: "Wait please"})
		handler.HandleUnclearInputExpired(context.Background(), expired)
		synctest.Wait()
		assert.Equal(t, contextID, requestor.GetID())
		for _, packet := range drainEgressPackets(requestor) {
			_, injected := packet.(internal_type.InjectMessagePacket)
			assert.False(t, injected)
		}
	})
}

func TestDispatchInterruptionNoWatchdogForOrdinaryListening(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat", adapter_lifecycle.WithInterruption(true))
		t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
		requestor.endOfSpeechExecutor = &recordingEOSExecutor{}
		turn, err := requestor.messageLifecycle.OnUserTurnStarted(requestor.GetID(), "test", "text", "hello")
		require.NoError(t, err)
		contextID := turn.ContextID
		streamer := requestor.streamer.(*streamTestStreamer)
		handler := requestorDispatchHandler{r: requestor}
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: contextID, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: contextID, Script: "hello", Interim: true})
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: contextID, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd})
		synctest.Wait()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		for _, packet := range drainEgressPackets(requestor) {
			_, expired := packet.(internal_type.UnclearInputExpiredPacket)
			assert.False(t, expired)
		}
		streamer.mu.Lock()
		assert.Empty(t, streamer.sent)
		streamer.mu.Unlock()
	})
}

func TestDispatchInterruptionPreservesWordTrigger(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerWord, adapter_lifecycle.WithInterruption(true))
		requestor.endOfSpeechExecutor = &recordingEOSExecutor{}
		streamer := requestor.streamer.(*streamTestStreamer)
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		assert.Equal(t, previous, requestor.GetID())
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "hello", Interim: true})
		synctest.Wait()
		assert.NotEqual(t, previous, requestor.GetID())
		streamer.mu.Lock()
		require.NotEmpty(t, streamer.sent)
		assert.Equal(t, protos.ConversationPlaybackControl_FLUSH, streamer.sent[0].(*protos.ConversationPlaybackControl).GetKind())
		streamer.mu.Unlock()
	})
}

func TestDispatchInterruptionInterimRefreshRejectsQueuedExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat", adapter_lifecycle.WithInterruption(true))
		t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
		requestor.endOfSpeechExecutor = &recordingEOSExecutor{}
		handler := requestorDispatchHandler{r: requestor}
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{Script: "wait", Interim: true})
		synctest.Wait()
		contextID := requestor.GetID()
		time.Sleep(time.Second)
		synctest.Wait()
		var expired internal_type.UnclearInputExpiredPacket
		for _, packet := range drainEgressPackets(requestor) {
			if value, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				expired = value
			}
		}
		require.Equal(t, contextID, expired.ContextID)
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: contextID, Script: "wait please", Interim: true})
		handler.HandleUnclearInputExpired(context.Background(), expired)
		synctest.Wait()
		assert.Equal(t, contextID, requestor.GetID())
		time.Sleep(999 * time.Millisecond)
		synctest.Wait()
		for _, packet := range drainEgressPackets(requestor) {
			_, expired := packet.(internal_type.UnclearInputExpiredPacket)
			assert.False(t, expired)
		}
		time.Sleep(time.Millisecond)
		synctest.Wait()
		var refreshed internal_type.UnclearInputExpiredPacket
		for _, packet := range drainEgressPackets(requestor) {
			if value, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				refreshed = value
			}
		}
		assert.Equal(t, contextID, refreshed.ContextID)
	})
}

func TestDispatchInterruptionFinalBeforeVadEndPreservesBoundary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat", adapter_lifecycle.WithInterruption(true))
		t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "Wait please"})
		synctest.Wait()
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd})
		synctest.Wait()
		packets := eos.snapshotExecuted()
		require.Len(t, packets, 4)
		boundary := packets[3].(internal_type.InterruptionDetectedPacket)
		assert.Equal(t, requestor.GetID(), boundary.ContextID)
		assert.Equal(t, internal_type.InterruptionEventEnd, boundary.Event)
		time.Sleep(2 * time.Second)
		synctest.Wait()
		for _, packet := range drainEgressPackets(requestor) {
			_, expired := packet.(internal_type.UnclearInputExpiredPacket)
			assert.False(t, expired)
		}
	})
}

func newInterruptionTestRequestor(trigger string, messageOptions ...adapter_lifecycle.MessageOption) *genericRequestor {
	options := map[string]interface{}{}
	if trigger != "" {
		options[internal_options.MicrophoneOptionBargeInTrigger] = trigger
	}

	requestorChannels := adapter_channel.NewRequestorChannels()
	r := &genericRequestor{
		streamer:      &streamTestStreamer{},
		channels:      requestorChannels,
		dispatchRoute: adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		options:       options,
		vadExecutor:   &blockingVADExecutor{},
	}
	lifecycleOptions := []adapter_lifecycle.MessageOption{
		adapter_lifecycle.WithContextID("ctx-active"),
		adapter_lifecycle.WithMode(type_enums.AudioMode),
		adapter_lifecycle.WithOnPacket(func(packets ...internal_type.Packet) error {
			return r.OnPacket(context.Background(), packets...)
		}),
		adapter_lifecycle.WithSend(func(message proto.Message) error {
			if r.streamer == nil {
				return fmt.Errorf("send %T: streamer is unavailable", message)
			}
			return r.streamer.Send(message)
		}),
		adapter_lifecycle.WithDispatch(requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket),
		adapter_lifecycle.WithInterruptionExpiry(func(packet internal_type.InterruptionDecisionExpiredPacket) {
			ctx := r.sessionCtx
			if ctx == nil {
				ctx = context.Background()
			}
			r.dispatch(ctx, packet)
		}),
		adapter_lifecycle.WithBehavior(r.deploymentBehavior),
	}
	r.messageLifecycle = adapter_lifecycle.NewMessageLifecycle(append(lifecycleOptions, messageOptions...)...)
	_ = r.messageLifecycle.OnGenerationStarted(r.GetID())
	_ = r.messageLifecycle.OnSpeechStarted(r.GetID())
	return r
}

func newUnclearInputTestRequestor(trigger string, timeout float64, message string, messageOptions ...adapter_lifecycle.MessageOption) *genericRequestor {
	r := newInterruptionTestRequestor(trigger, messageOptions...)
	r.source = utils.PhoneCall
	r.assistant = &internal_assistant_entity.Assistant{
		AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				UnclearInputTimeout: &timeout,
				UnclearInputMessage: &message,
			},
		},
	}
	if err := r.messageLifecycle.Initialize(context.Background()); err != nil {
		panic(err)
	}
	drainBackgroundPackets(r)
	return r
}

func drainControlPackets(r *genericRequestor) []internal_type.Packet {
	packets := make([]internal_type.Packet, 0, r.channels.ControlChannel().Len())
	for r.channels.ControlChannel().Len() > 0 {
		envelope, err := r.channels.ControlChannel().TryReceive()
		if err != nil {
			break
		}
		packets = append(packets, envelope.Pkt)
	}
	return packets
}

func drainEgressPackets(r *genericRequestor) []internal_type.Packet {
	packets := make([]internal_type.Packet, 0, r.channels.EgressChannel().Len())
	for r.channels.EgressChannel().Len() > 0 {
		envelope, err := r.channels.EgressChannel().TryReceive()
		if err != nil {
			break
		}
		packets = append(packets, envelope.Pkt)
	}
	return packets
}

func drainIngressPackets(r *genericRequestor) []internal_type.Packet {
	packets := make([]internal_type.Packet, 0, r.channels.IngressChannel().Len())
	for r.channels.IngressChannel().Len() > 0 {
		envelope, err := r.channels.IngressChannel().TryReceive()
		if err != nil {
			break
		}
		packets = append(packets, envelope.Pkt)
	}
	return packets
}

func drainBackgroundPackets(r *genericRequestor) []internal_type.Packet {
	packets := make([]internal_type.Packet, 0, r.channels.BackgroundChannel().Len())
	for r.channels.BackgroundChannel().Len() > 0 {
		envelope, err := r.channels.BackgroundChannel().TryReceive()
		if err != nil {
			break
		}
		packets = append(packets, envelope.Pkt)
	}
	return packets
}

func waitForUnclearInputExpired(t *testing.T, r *genericRequestor) internal_type.UnclearInputExpiredPacket {
	t.Helper()

	var expired internal_type.UnclearInputExpiredPacket
	require.Eventually(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if typed, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				expired = typed
			}
		}
		return expired.ContextID != ""
	}, time.Second, 10*time.Millisecond)
	return expired
}

func TestHandleUserText_TextModeRotatesWordInterruptionBeforeEOS(t *testing.T) {
	r := &genericRequestor{
		streamer:    &streamTestStreamer{},
		channels:    adapter_channel.NewRequestorChannels(),
		vadExecutor: &blockingVADExecutor{},
	}
	lifecycle := adapter_lifecycle.NewMessageLifecycle(
		adapter_lifecycle.WithContextID("ctx-active"),
		adapter_lifecycle.WithMode(type_enums.TextMode),
		adapter_lifecycle.WithSend(func(message proto.Message) error { return r.streamer.Send(message) }),
		adapter_lifecycle.WithDispatch(requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket),
	)
	r.messageLifecycle = lifecycle
	require.NoError(t, lifecycle.OnGenerationStarted("ctx-active"))
	require.NoError(t, lifecycle.OnSpeechStarted("ctx-active"))

	h := requestorDispatchHandler{r: r}

	h.HandleUserText(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-active",
		Text:      "interrupt with text",
	})

	newContextID := r.GetID()
	require.NotEqual(t, "ctx-active", newContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())

	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range drainControlPackets(r) {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, "ctx-active", eosInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceWord, eosInterrupt.Source)
	assert.Equal(t, "ctx-active", ttsInterrupt.ContextID)
	assert.Equal(t, "ctx-active", llmInterrupt.ContextID)

	var stopIdleTimeout internal_type.StopIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StopIdleTimeoutPacket); ok {
			stopIdleTimeout = typed
		}
	}
	assert.Equal(t, "ctx-active", stopIdleTimeout.ContextID)

	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 2)
	interim, ok := ingressPackets[0].(internal_type.InterimEndOfSpeechPacket)
	require.True(t, ok, "expected InterimEndOfSpeechPacket, got %T", ingressPackets[0])
	assert.Equal(t, newContextID, interim.ContextID)
	assert.Equal(t, "interrupt with text", interim.Speech)
	eos, ok := ingressPackets[1].(internal_type.EndOfSpeechPacket)
	require.True(t, ok, "expected EndOfSpeechPacket, got %T", ingressPackets[1])
	assert.Equal(t, newContextID, eos.ContextID)
	assert.Equal(t, "interrupt with text", eos.Speech)
}

func TestHandleUserText_TextModeRotatesAfterPreviousUserFinished(t *testing.T) {
	r := &genericRequestor{
		streamer:    &streamTestStreamer{},
		channels:    adapter_channel.NewRequestorChannels(),
		vadExecutor: &blockingVADExecutor{},
	}
	lifecycle := adapter_lifecycle.NewMessageLifecycle(
		adapter_lifecycle.WithContextID("ctx-first"),
		adapter_lifecycle.WithMode(type_enums.TextMode),
		adapter_lifecycle.WithSend(func(message proto.Message) error { return r.streamer.Send(message) }),
		adapter_lifecycle.WithDispatch(requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket),
	)
	r.messageLifecycle = lifecycle
	require.NoError(t, lifecycle.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: "ctx-first", Speech: "first text"}))

	h := requestorDispatchHandler{r: r}

	h.HandleUserText(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-first",
		Text:      "second text",
	})

	secondContextID := r.GetID()
	require.NotEqual(t, "ctx-first", secondContextID)

	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 2)
	interim, ok := ingressPackets[0].(internal_type.InterimEndOfSpeechPacket)
	require.True(t, ok, "expected InterimEndOfSpeechPacket, got %T", ingressPackets[0])
	assert.Equal(t, secondContextID, interim.ContextID)
	assert.Equal(t, "second text", interim.Speech)
	eos, ok := ingressPackets[1].(internal_type.EndOfSpeechPacket)
	require.True(t, ok, "expected EndOfSpeechPacket, got %T", ingressPackets[1])
	assert.Equal(t, secondContextID, eos.ContextID)
	assert.Equal(t, "second text", eos.Speech)
}

func TestHandleUserText_TextModeDoesNotRotateWhileUserSpeaking(t *testing.T) {
	r := &genericRequestor{
		streamer:    &streamTestStreamer{},
		channels:    adapter_channel.NewRequestorChannels(),
		vadExecutor: &blockingVADExecutor{},
	}
	lifecycle := adapter_lifecycle.NewMessageLifecycle(
		adapter_lifecycle.WithContextID("ctx-user-speaking"),
		adapter_lifecycle.WithMode(type_enums.TextMode),
		adapter_lifecycle.WithSend(func(message proto.Message) error { return r.streamer.Send(message) }),
		adapter_lifecycle.WithDispatch(requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket),
	)
	r.messageLifecycle = lifecycle
	lifecycle.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-user-speaking", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}, internal_options.BargeInTriggerVAD)
	contextID := lifecycle.ContextID()
	require.Equal(t, adapter_lifecycle.MessageStateUserSpeaking, lifecycle.State())

	h := requestorDispatchHandler{r: r}

	h.HandleUserText(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: contextID,
		Text:      "same turn text",
	})

	assert.Equal(t, contextID, r.GetID())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))

	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 2)
	interim, ok := ingressPackets[0].(internal_type.InterimEndOfSpeechPacket)
	require.True(t, ok, "expected InterimEndOfSpeechPacket, got %T", ingressPackets[0])
	assert.Equal(t, contextID, interim.ContextID)
	eos, ok := ingressPackets[1].(internal_type.EndOfSpeechPacket)
	require.True(t, ok, "expected EndOfSpeechPacket, got %T", ingressPackets[1])
	assert.Equal(t, contextID, eos.ContextID)
}

func TestHandleUserText_TextModeDoesNotRotateWhenUserTurnActive(t *testing.T) {
	r := &genericRequestor{
		streamer:    &streamTestStreamer{},
		channels:    adapter_channel.NewRequestorChannels(),
		vadExecutor: &blockingVADExecutor{},
	}
	lifecycle := adapter_lifecycle.NewMessageLifecycle(
		adapter_lifecycle.WithContextID("ctx-listening"),
		adapter_lifecycle.WithMode(type_enums.TextMode),
		adapter_lifecycle.WithSend(func(message proto.Message) error { return r.streamer.Send(message) }),
		adapter_lifecycle.WithDispatch(requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket),
	)
	r.messageLifecycle = lifecycle
	_, err := lifecycle.OnTranscriptReceived(internal_type.SpeechToTextPacket{ContextID: "ctx-listening", Script: "same turn text"})
	require.NoError(t, err)

	h := requestorDispatchHandler{r: r}

	h.HandleUserText(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-listening",
		Text:      "same turn text",
	})

	assert.Equal(t, "ctx-listening", r.GetID())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))

	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 2)
	interim, ok := ingressPackets[0].(internal_type.InterimEndOfSpeechPacket)
	require.True(t, ok, "expected InterimEndOfSpeechPacket, got %T", ingressPackets[0])
	assert.Equal(t, "ctx-listening", interim.ContextID)
	eos, ok := ingressPackets[1].(internal_type.EndOfSpeechPacket)
	require.True(t, ok, "expected EndOfSpeechPacket, got %T", ingressPackets[1])
	assert.Equal(t, "ctx-listening", eos.ContextID)
}

func TestHandleInterruptionDetected_TextModeIgnoresStaleWordInterruption(t *testing.T) {
	r := &genericRequestor{
		streamer:    &streamTestStreamer{},
		channels:    adapter_channel.NewRequestorChannels(),
		vadExecutor: &blockingVADExecutor{},
	}
	lifecycle := adapter_lifecycle.NewMessageLifecycle(
		adapter_lifecycle.WithContextID("ctx-current"),
		adapter_lifecycle.WithMode(type_enums.TextMode),
		adapter_lifecycle.WithSend(func(message proto.Message) error { return r.streamer.Send(message) }),
		adapter_lifecycle.WithDispatch(requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket),
	)
	r.messageLifecycle = lifecycle
	require.NoError(t, lifecycle.OnGenerationStarted("ctx-current"))
	require.NoError(t, lifecycle.OnSpeechStarted("ctx-current"))

	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-stale",
		Source:    internal_type.InterruptionSourceWord,
	})

	assert.Equal(t, "ctx-current", r.GetID())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))
	assert.Empty(t, drainIngressPackets(r))
}

func TestHandleUserInput_DropsStaleContext(t *testing.T) {
	r := &genericRequestor{
		streamer:    &streamTestStreamer{},
		channels:    adapter_channel.NewRequestorChannels(),
		vadExecutor: &blockingVADExecutor{},
	}
	lifecycle := adapter_lifecycle.NewMessageLifecycle(
		adapter_lifecycle.WithContextID("ctx-current"),
		adapter_lifecycle.WithMode(type_enums.TextMode),
		adapter_lifecycle.WithSend(func(message proto.Message) error { return r.streamer.Send(message) }),
		adapter_lifecycle.WithDispatch(requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket),
	)
	r.messageLifecycle = lifecycle

	h := requestorDispatchHandler{r: r}

	h.HandleUserInput(context.Background(), internal_type.UserInputPacket{
		ContextID: "ctx-stale",
		Text:      "old input",
	})

	assert.Equal(t, "ctx-current", r.GetID())
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))
	assert.Empty(t, drainIngressPackets(r))
}

func TestHandleSpeechToText_FinalRotatesAfterPreviousUserFinished(t *testing.T) {
	r := &genericRequestor{
		streamer:    &streamTestStreamer{},
		channels:    adapter_channel.NewRequestorChannels(),
		vadExecutor: &blockingVADExecutor{},
	}
	lifecycle := adapter_lifecycle.NewMessageLifecycle(
		adapter_lifecycle.WithContextID("ctx-first"),
		adapter_lifecycle.WithMode(type_enums.AudioMode),
		adapter_lifecycle.WithSend(func(message proto.Message) error { return r.streamer.Send(message) }),
		adapter_lifecycle.WithDispatch(requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket),
	)
	r.messageLifecycle = lifecycle
	require.NoError(t, lifecycle.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: "ctx-first", Speech: "first turn"}))

	h := requestorDispatchHandler{r: r}

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-first",
		Script:    "new audio turn",
		Interim:   false,
	})

	newContextID := r.GetID()
	require.NotEqual(t, "ctx-first", newContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())

	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range drainControlPackets(r) {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, "ctx-first", eosInterrupt.ContextID)
	assert.Equal(t, "ctx-first", ttsInterrupt.ContextID)
	assert.Equal(t, "ctx-first", llmInterrupt.ContextID)

	var stopIdleTimeout internal_type.StopIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StopIdleTimeoutPacket); ok {
			stopIdleTimeout = typed
		}
	}
	assert.Equal(t, "ctx-first", stopIdleTimeout.ContextID)

	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 1)
	eos, ok := ingressPackets[0].(internal_type.EndOfSpeechPacket)
	require.True(t, ok, "expected EndOfSpeechPacket, got %T", ingressPackets[0])
	assert.Equal(t, newContextID, eos.ContextID)
	assert.Equal(t, "new audio turn", eos.Speech)

	h.HandleEndOfSpeech(context.Background(), eos)
	assert.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
}

func TestHandleSpeechToText_DoesNotRotateAgainForSameSpeechSegment(t *testing.T) {
	r := &genericRequestor{
		streamer:    &streamTestStreamer{},
		channels:    adapter_channel.NewRequestorChannels(),
		vadExecutor: &blockingVADExecutor{},
	}
	lifecycle := adapter_lifecycle.NewMessageLifecycle(
		adapter_lifecycle.WithContextID("ctx-first"),
		adapter_lifecycle.WithMode(type_enums.AudioMode),
		adapter_lifecycle.WithSend(func(message proto.Message) error { return r.streamer.Send(message) }),
		adapter_lifecycle.WithDispatch(requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket),
	)
	r.messageLifecycle = lifecycle
	require.NoError(t, lifecycle.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: "ctx-first", Speech: "first turn"}))

	executor := &recordingEOSExecutor{}
	r.endOfSpeechExecutor = executor
	h := requestorDispatchHandler{r: r}

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-first",
		Script:    "Yes. I am.",
		Interim:   false,
	})

	turnContextID := r.GetID()
	require.NotEqual(t, "ctx-first", turnContextID)

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-first",
		Script:    "Yeah.",
		Interim:   false,
	})

	assert.Equal(t, turnContextID, r.GetID())
	executed := executor.snapshotExecuted()
	require.Len(t, executed, 2)
	first, ok := executed[0].(internal_type.SpeechToTextPacket)
	require.True(t, ok)
	second, ok := executed[1].(internal_type.SpeechToTextPacket)
	require.True(t, ok)
	assert.Equal(t, turnContextID, first.ContextID)
	assert.Equal(t, turnContextID, second.ContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())
}

func TestHandleSpeechToText_DropsStaleTranscriptAfterUserFinished(t *testing.T) {
	r := &genericRequestor{
		streamer:    &streamTestStreamer{},
		channels:    adapter_channel.NewRequestorChannels(),
		vadExecutor: &blockingVADExecutor{},
	}
	lifecycle := adapter_lifecycle.NewMessageLifecycle(
		adapter_lifecycle.WithContextID("ctx-current"),
		adapter_lifecycle.WithMode(type_enums.AudioMode),
		adapter_lifecycle.WithSend(func(message proto.Message) error { return r.streamer.Send(message) }),
		adapter_lifecycle.WithDispatch(requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket),
	)
	r.messageLifecycle = lifecycle
	require.NoError(t, lifecycle.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: "ctx-current", Speech: "current turn"}))

	h := requestorDispatchHandler{r: r}

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-old",
		Script:    "late transcript",
		Interim:   false,
	})

	assert.Equal(t, "ctx-current", r.GetID())
	assert.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
	assert.Empty(t, drainControlPackets(r))
	assert.Empty(t, drainEgressPackets(r))
	assert.Empty(t, drainIngressPackets(r))
}

func TestHandleInterimEndOfSpeech_UsesPacketContextID(t *testing.T) {
	streamer := &streamTestStreamer{}
	r := &genericRequestor{
		streamer:         streamer,
		channels:         adapter_channel.NewRequestorChannels(),
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(adapter_lifecycle.WithContextID("ctx-current"), adapter_lifecycle.WithMode(type_enums.AudioMode)),
		vadExecutor:      &blockingVADExecutor{},
	}
	h := requestorDispatchHandler{r: r}

	h.HandleInterimEndOfSpeech(context.Background(), internal_type.InterimEndOfSpeechPacket{
		ContextID: "ctx-segment",
		Speech:    "partial text",
	})

	require.Len(t, streamer.sent, 1)
	userMessage, ok := streamer.sent[0].(*protos.ConversationUserMessage)
	require.True(t, ok, "expected ConversationUserMessage, got %T", streamer.sent[0])
	assert.Equal(t, "ctx-segment", userMessage.Id)
	assert.False(t, userMessage.Completed)
}

func TestHandleInterruptionDetected_VADTriggerUsesVADOnly(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.2, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceWord,
	})

	assert.Equal(t, "ctx-active", r.GetID())
	assert.Zero(t, r.channels.ControlChannel().Len())
	assert.Zero(t, r.channels.EgressChannel().Len())

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	userTurnContextID := r.GetID()
	require.NotEqual(t, "ctx-active", userTurnContextID)
	controlPackets := drainControlPackets(r)
	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	var sttStart internal_type.SpeechToTextStartPacket
	for _, packet := range controlPackets {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		case internal_type.SpeechToTextStartPacket:
			sttStart = typed
		}
	}
	assert.Equal(t, "ctx-active", eosInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceVad, eosInterrupt.Source)
	assert.Equal(t, "ctx-active", ttsInterrupt.ContextID)
	assert.Equal(t, "ctx-active", llmInterrupt.ContextID)
	assert.Equal(t, userTurnContextID, sttStart.ContextID)

	var stopIdleTimeout internal_type.StopIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StopIdleTimeoutPacket); ok {
			stopIdleTimeout = typed
		}
	}
	assert.Equal(t, "ctx-active", stopIdleTimeout.ContextID)

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})

	var sttEnd internal_type.SpeechToTextEndPacket
	for _, packet := range drainControlPackets(r) {
		if typed, ok := packet.(internal_type.SpeechToTextEndPacket); ok {
			sttEnd = typed
		}
	}
	require.NotEmpty(t, sttEnd.ContextID)
	assert.Equal(t, userTurnContextID, sttEnd.ContextID)
}

func TestHandleInterruptionDetected_WordTriggerUsesWordOnly(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerWord, 0.2, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	assert.Equal(t, "ctx-active", r.GetID())
	controlPackets := drainControlPackets(r)
	require.Len(t, controlPackets, 1)
	sttStart, ok := controlPackets[0].(internal_type.SpeechToTextStartPacket)
	require.True(t, ok, "expected SpeechToTextStartPacket, got %T", controlPackets[0])
	assert.Equal(t, "ctx-active", sttStart.ContextID)
	assert.Zero(t, r.channels.EgressChannel().Len())

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceWord,
	})

	userTurnContextID := r.GetID()
	require.NotEqual(t, "ctx-active", userTurnContextID)
	controlPackets = drainControlPackets(r)
	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range controlPackets {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, "ctx-active", eosInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceWord, eosInterrupt.Source)
	assert.Equal(t, "ctx-active", ttsInterrupt.ContextID)
	assert.Equal(t, "ctx-active", llmInterrupt.ContextID)

	var stopIdleTimeout internal_type.StopIdleTimeoutPacket
	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StopIdleTimeoutPacket); ok {
			stopIdleTimeout = typed
		}
	}
	assert.Equal(t, "ctx-active", stopIdleTimeout.ContextID)

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})

	var sttEnd internal_type.SpeechToTextEndPacket
	for _, packet := range drainControlPackets(r) {
		if typed, ok := packet.(internal_type.SpeechToTextEndPacket); ok {
			sttEnd = typed
		}
	}
	require.NotEmpty(t, sttEnd.ContextID)
	assert.Equal(t, userTurnContextID, sttEnd.ContextID)
}

func TestHandleInterruptionDetected_VADTriggerStartsUnclearInputWatchdogAfterVADEnd(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.02, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	drainEgressPackets(r)
	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 40*time.Millisecond, 10*time.Millisecond)

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})

	expired := waitForUnclearInputExpired(t, r)
	assert.Equal(t, r.GetID(), expired.ContextID)
}

func TestHandleInterruptionDetected_WordTriggerStartsUnclearInputWatchdogOnlyAfterWord(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerWord, 0.02, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	select {
	case <-r.channels.EgressChannel().Ready():
		packet := receiveEnvelope(t, r.channels.EgressChannel())
		t.Fatalf("unclear input watchdog started before word interruption: %+v", packet.Pkt)
	case <-time.After(50 * time.Millisecond):
	}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceWord,
	})

	expired := waitForUnclearInputExpired(t, r)
	assert.Equal(t, r.GetID(), expired.ContextID)
}

func TestHandleInterruptionDetected_WordTriggerDuplicateWordExtendsUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerWord, 0.06, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: contextID,
		Source:    internal_type.InterruptionSourceWord,
	})
	drainEgressPackets(r)

	time.Sleep(35 * time.Millisecond)
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: contextID,
		Source:    internal_type.InterruptionSourceWord,
	})

	select {
	case <-r.channels.EgressChannel().Ready():
		packet := receiveEnvelope(t, r.channels.EgressChannel())
		t.Fatalf("unclear input watchdog expired before duplicate word extended deadline: %+v", packet.Pkt)
	case <-time.After(35 * time.Millisecond):
	}

	expired := waitForUnclearInputExpired(t, r)
	assert.Equal(t, r.GetID(), expired.ContextID)
}

func TestHandleInterruptionDetected_WordTriggerDuplicateAfterFinalDoesNotRestartUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerWord, 0.03, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceWord,
	})
	drainEgressPackets(r)
	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hello",
		Interim:   false,
	})

	newContextID := r.GetID()
	require.NotEqual(t, oldContextID, newContextID)

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceWord,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: newContextID,
		Source:    internal_type.InterruptionSourceWord,
	})

	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 80*time.Millisecond, 10*time.Millisecond)
}

func TestHandleSpeechToText_InterimStopsUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.06, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	drainEgressPackets(r)

	time.Sleep(35 * time.Millisecond)
	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hel",
		Interim:   true,
	})

	newContextID := r.GetID()
	require.NotEqual(t, oldContextID, newContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())

	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 80*time.Millisecond, 10*time.Millisecond)
}

func TestHandleEndOfSpeech_StopsUnclearInputWatchdogForAcceptedSpeech(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-active",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	drainEgressPackets(r)
	turnContextID := r.GetID()
	require.NotEqual(t, "ctx-active", turnContextID)
	drainControlPackets(r)

	h.HandleEndOfSpeech(context.Background(), internal_type.EndOfSpeechPacket{
		ContextID: turnContextID,
		Speech:    "hello",
	})

	assert.Equal(t, turnContextID, r.GetID())
	assert.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
	for _, packet := range drainControlPackets(r) {
		switch packet.(type) {
		case internal_type.TextToSpeechInterruptPacket, internal_type.LLMInterruptPacket:
			t.Fatalf("unexpected interrupt packet from EOS completion: %T", packet)
		}
	}

	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 80*time.Millisecond, 10*time.Millisecond)
}

func TestHandleEndOfSpeech_FinalizesWhenEOSCompletesDuringVADSpeaking(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	turnContextID := r.GetID()
	require.NotEqual(t, oldContextID, turnContextID)
	drainControlPackets(r)
	drainEgressPackets(r)

	h.HandleEndOfSpeech(context.Background(), internal_type.EndOfSpeechPacket{
		ContextID: turnContextID,
		Speech:    "still speaking",
	})

	assert.Equal(t, turnContextID, r.GetID())
	assert.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 1)
	userInput, ok := ingressPackets[0].(internal_type.UserInputPacket)
	require.True(t, ok, "expected UserInputPacket, got %T", ingressPackets[0])
	assert.Equal(t, turnContextID, userInput.ContextID)
	assert.Equal(t, "still speaking", userInput.Text)
}

func TestHandleEndOfSpeech_FinalizesDuringVADSpeakingWithoutWatchdog(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})

	turnContextID := r.GetID()
	require.NotEqual(t, oldContextID, turnContextID)
	drainControlPackets(r)
	drainEgressPackets(r)

	h.HandleEndOfSpeech(context.Background(), internal_type.EndOfSpeechPacket{
		ContextID: turnContextID,
		Speech:    "fallback speech",
	})

	assert.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
	ingressPackets := drainIngressPackets(r)
	require.Len(t, ingressPackets, 1)
	userInput, ok := ingressPackets[0].(internal_type.UserInputPacket)
	require.True(t, ok, "expected UserInputPacket, got %T", ingressPackets[0])
	assert.Equal(t, turnContextID, userInput.ContextID)
	assert.Equal(t, "fallback speech", userInput.Text)
}

func TestHandleSpeechToText_FinalStopsUnclearInputWatchdog(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	drainEgressPackets(r)
	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hello",
		Interim:   false,
	})

	newContextID := r.GetID()
	require.NotEqual(t, oldContextID, newContextID)

	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range drainControlPackets(r) {
		switch typed := packet.(type) {
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, oldContextID, ttsInterrupt.ContextID)
	assert.Equal(t, oldContextID, llmInterrupt.ContextID)

	var eos internal_type.EndOfSpeechPacket
	for _, packet := range drainIngressPackets(r) {
		if typed, ok := packet.(internal_type.EndOfSpeechPacket); ok {
			eos = typed
		}
	}
	require.Equal(t, "hello", eos.Speech)
	assert.Equal(t, newContextID, eos.ContextID)
	h.HandleEndOfSpeech(context.Background(), eos)

	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 80*time.Millisecond, 10*time.Millisecond)
}

func TestHandleSpeechToText_InterimStartsTurnAndFinalKeepsContext(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.06, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	drainEgressPackets(r)

	time.Sleep(35 * time.Millisecond)
	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hel",
		Interim:   true,
	})

	newContextID := r.GetID()
	require.NotEqual(t, oldContextID, newContextID)

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hello",
		Interim:   false,
	})

	assert.Equal(t, newContextID, r.GetID())

	require.Never(t, func() bool {
		for _, packet := range drainEgressPackets(r) {
			if _, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				return true
			}
		}
		return false
	}, 80*time.Millisecond, 10*time.Millisecond)
}

func TestHandleUnclearInputExpired_InjectsConfiguredMessage(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	turn, err := r.messageLifecycle.OnUserTurnStarted(r.GetID(), "test", "text", "unclear input")
	require.NoError(t, err)
	contextID, err := r.messageLifecycle.OnTranscriptReceived(internal_type.SpeechToTextPacket{ContextID: turn.ContextID, Script: "unclear input", Interim: true})
	require.NoError(t, err)

	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: contextID})

	newContextID := r.GetID()
	require.NotEqual(t, contextID, newContextID)

	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	for _, packet := range drainControlPackets(r) {
		if typed, ok := packet.(internal_type.TextToSpeechInterruptPacket); ok {
			ttsInterrupt = typed
		}
	}

	var injectMessage internal_type.InjectMessagePacket
	for _, packet := range drainEgressPackets(r) {
		switch typed := packet.(type) {
		case internal_type.InjectMessagePacket:
			injectMessage = typed
		case internal_type.StartIdleTimeoutPacket:
			t.Fatalf("unclear prompt should not restart idle timer before assistant completion: %+v", typed)
		}
	}

	assert.Equal(t, contextID, ttsInterrupt.ContextID)
	assert.Equal(t, newContextID, injectMessage.ContextID)
	assert.Equal(t, "Please say that again.", injectMessage.Text)
}

func TestHandleUnclearInputExpired_VADTriggerRotatesFromInterruptedContextAndInjectsNewPrompt(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "I didn't catch that.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	controlPackets := drainControlPackets(r)
	var startEOSInterrupt internal_type.EndOfSpeechInterruptionPacket
	var sttStart internal_type.SpeechToTextStartPacket
	for _, packet := range controlPackets {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			startEOSInterrupt = typed
		case internal_type.SpeechToTextStartPacket:
			sttStart = typed
		}
	}
	userTurnContextID := r.GetID()
	require.NotEqual(t, oldContextID, userTurnContextID)
	assert.Equal(t, oldContextID, startEOSInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceVad, startEOSInterrupt.Source)
	assert.Equal(t, userTurnContextID, sttStart.ContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateUserSpeaking, r.messageLifecycle.State())

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	controlPackets = drainControlPackets(r)
	require.Len(t, controlPackets, 1)
	sttEnd, ok := controlPackets[0].(internal_type.SpeechToTextEndPacket)
	require.True(t, ok)
	assert.Equal(t, userTurnContextID, sttEnd.ContextID)
	assert.Equal(t, userTurnContextID, r.GetID())
	assert.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())

	drainEgressPackets(r)
	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: userTurnContextID})

	promptContextID := r.GetID()
	require.NotEqual(t, userTurnContextID, promptContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())

	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range drainControlPackets(r) {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, userTurnContextID, eosInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceVad, eosInterrupt.Source)
	assert.Equal(t, userTurnContextID, ttsInterrupt.ContextID)
	assert.Equal(t, userTurnContextID, llmInterrupt.ContextID)

	var stopIdleTimeout internal_type.StopIdleTimeoutPacket
	var injectMessage internal_type.InjectMessagePacket
	var prompts []internal_type.InjectMessagePacket
	for _, packet := range drainEgressPackets(r) {
		switch typed := packet.(type) {
		case internal_type.StopIdleTimeoutPacket:
			stopIdleTimeout = typed
		case internal_type.InjectMessagePacket:
			injectMessage = typed
			prompts = append(prompts, typed)
		case internal_type.StartIdleTimeoutPacket:
			t.Fatalf("unclear prompt should not start idle before assistant completion: %+v", typed)
		}
	}
	assert.Len(t, prompts, 1)
	assert.Equal(t, userTurnContextID, stopIdleTimeout.ContextID)
	assert.Equal(t, promptContextID, injectMessage.ContextID)
	assert.Equal(t, "I didn't catch that.", injectMessage.Text)
}

func TestHandleUnclearInputExpired_WordTriggerRotatesFromInterruptedContextAndInjectsNewPrompt(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerWord, 0.03, "I didn't catch that.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	controlPackets := drainControlPackets(r)
	require.Len(t, controlPackets, 1)
	sttStart, ok := controlPackets[0].(internal_type.SpeechToTextStartPacket)
	require.True(t, ok)
	assert.Equal(t, oldContextID, sttStart.ContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, r.messageLifecycle.State())

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceWord,
	})
	userTurnContextID := r.GetID()
	require.NotEqual(t, oldContextID, userTurnContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())
	controlPackets = drainControlPackets(r)
	var wordEOSInterrupt internal_type.EndOfSpeechInterruptionPacket
	for _, packet := range controlPackets {
		if typed, ok := packet.(internal_type.EndOfSpeechInterruptionPacket); ok {
			wordEOSInterrupt = typed
		}
	}
	assert.Equal(t, oldContextID, wordEOSInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceWord, wordEOSInterrupt.Source)

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: oldContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})
	controlPackets = drainControlPackets(r)
	require.Len(t, controlPackets, 1)
	sttEnd, ok := controlPackets[0].(internal_type.SpeechToTextEndPacket)
	require.True(t, ok)
	assert.Equal(t, userTurnContextID, sttEnd.ContextID)

	drainEgressPackets(r)
	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: userTurnContextID})

	promptContextID := r.GetID()
	require.NotEqual(t, userTurnContextID, promptContextID)
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())

	var eosInterrupt internal_type.EndOfSpeechInterruptionPacket
	var ttsInterrupt internal_type.TextToSpeechInterruptPacket
	var llmInterrupt internal_type.LLMInterruptPacket
	for _, packet := range drainControlPackets(r) {
		switch typed := packet.(type) {
		case internal_type.EndOfSpeechInterruptionPacket:
			eosInterrupt = typed
		case internal_type.TextToSpeechInterruptPacket:
			ttsInterrupt = typed
		case internal_type.LLMInterruptPacket:
			llmInterrupt = typed
		}
	}
	assert.Equal(t, userTurnContextID, eosInterrupt.ContextID)
	assert.Equal(t, internal_type.InterruptionSourceWord, eosInterrupt.Source)
	assert.Equal(t, userTurnContextID, ttsInterrupt.ContextID)
	assert.Equal(t, userTurnContextID, llmInterrupt.ContextID)

	var injectMessage internal_type.InjectMessagePacket
	var prompts []internal_type.InjectMessagePacket
	for _, packet := range drainEgressPackets(r) {
		switch typed := packet.(type) {
		case internal_type.InjectMessagePacket:
			injectMessage = typed
			prompts = append(prompts, typed)
		case internal_type.StartIdleTimeoutPacket:
			t.Fatalf("unclear prompt should not start idle before assistant completion: %+v", typed)
		}
	}
	assert.Len(t, prompts, 1)
	assert.Equal(t, promptContextID, injectMessage.ContextID)
	assert.Equal(t, "I didn't catch that.", injectMessage.Text)
}

func TestHandleInterruptionDetected_ForwardsVADEndToEOSWhenLifecycleAlreadyListening(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	executor := &recordingEOSExecutor{}
	r.endOfSpeechExecutor = executor
	h := requestorDispatchHandler{r: r}
	oldContextID := r.GetID()

	h.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: oldContextID,
		Script:    "hello",
		Interim:   false,
	})
	turnContextID := r.GetID()
	require.NotEqual(t, oldContextID, turnContextID)
	require.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())

	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: turnContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventStart,
	})
	h.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: turnContextID,
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	})

	controlPackets := drainControlPackets(r)
	var sttEnd internal_type.SpeechToTextEndPacket
	for _, packet := range controlPackets {
		if typed, ok := packet.(internal_type.SpeechToTextEndPacket); ok {
			sttEnd = typed
		}
	}
	assert.Equal(t, turnContextID, sttEnd.ContextID)

	executed := executor.snapshotExecuted()
	require.Len(t, executed, 3)
	_, ok := executed[0].(internal_type.SpeechToTextPacket)
	require.True(t, ok, "expected STT packet, got %T", executed[0])
	vadStart, ok := executed[1].(internal_type.InterruptionDetectedPacket)
	require.True(t, ok, "expected VAD start packet, got %T", executed[1])
	assert.Equal(t, internal_type.InterruptionEventStart, vadStart.Event)
	vadEnd, ok := executed[2].(internal_type.InterruptionDetectedPacket)
	require.True(t, ok, "expected VAD end packet, got %T", executed[2])
	assert.Equal(t, internal_type.InterruptionEventEnd, vadEnd.Event)
	assert.Equal(t, turnContextID, vadEnd.ContextID)
}

func TestHandleUnclearInputExpired_IgnoresWhenInterruptionIsNotPending(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()
	r.messageLifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: contextID, Text: "done"})
	require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
		Id: contextID, Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "done"},
	}))
	require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
		Id: contextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
	}))
	require.NoError(t, r.messageLifecycle.OnPlaybackCompleted(contextID))
	require.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())

	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: contextID})

	assert.Equal(t, contextID, r.GetID())
	assert.Empty(t, drainControlPackets(r))
	assert.Equal(t, []internal_type.Packet{internal_type.StartIdleTimeoutPacket{ContextID: contextID}}, drainEgressPackets(r))
}

func TestHandleIdleTimeoutExpired_InterruptsOldContextAndInjectsOnNewContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		idleTimeout := uint64(10)
		message := "Are you still there?"
		r.source = utils.PhoneCall
		r.assistant = &internal_assistant_entity.Assistant{
			AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
				AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
					IdleTimeout:        &idleTimeout,
					IdleTimeoutMessage: &message,
				},
			},
		}
		r.sessionLifecycle = adapter_lifecycle.NewSessionLifecycle()
		r.sessionLifecycle.ConfigureTimeouts(context.Background(), r.GetID(), &r.assistant.AssistantPhoneDeployment.AssistantDeploymentBehavior, r.OnPacket)
		t.Cleanup(r.sessionLifecycle.CloseTimeouts)
		h := requestorDispatchHandler{r: r}
		oldContextID := r.GetID()
		r.messageLifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: oldContextID, Text: "done"})
		require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
			Id: oldContextID, Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "done"},
		}))
		require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
			Id: oldContextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
		}))
		require.NoError(t, r.messageLifecycle.OnPlaybackCompleted(oldContextID))
		require.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())

		h.HandleStartIdleTimeout(context.Background(), internal_type.StartIdleTimeoutPacket{ContextID: oldContextID})
		time.Sleep(10 * time.Second)
		synctest.Wait()
		var idleTimeoutExpired internal_type.IdleTimeoutExpiredPacket
		for _, packet := range drainEgressPackets(r) {
			if expired, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
				idleTimeoutExpired = expired
			}
		}
		require.NotZero(t, idleTimeoutExpired.Generation)
		h.HandleIdleTimeoutExpired(context.Background(), idleTimeoutExpired)

		newContextID := r.GetID()
		require.NotEqual(t, oldContextID, newContextID)

		var ttsInterrupt internal_type.TextToSpeechInterruptPacket
		for _, packet := range drainControlPackets(r) {
			if typed, ok := packet.(internal_type.TextToSpeechInterruptPacket); ok {
				ttsInterrupt = typed
			}
		}

		var injectMessage internal_type.InjectMessagePacket
		var prompts []internal_type.InjectMessagePacket
		for _, packet := range drainEgressPackets(r) {
			switch typed := packet.(type) {
			case internal_type.InjectMessagePacket:
				injectMessage = typed
				prompts = append(prompts, typed)
			case internal_type.StartIdleTimeoutPacket:
				t.Fatalf("idle timeout prompt should not restart idle timer before assistant completion: %+v", typed)
			}
		}

		assert.Len(t, prompts, 1)
		assert.Equal(t, oldContextID, ttsInterrupt.ContextID)
		assert.Equal(t, newContextID, injectMessage.ContextID)
		assert.Equal(t, message, injectMessage.Text)
	})
}

func TestHandleIdleTimeoutExpired_InjectedPromptSpeaksBeforeIdleRestarts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		idleTimeout := uint64(10)
		message := "Are you still there?"
		r.source = utils.PhoneCall
		r.assistant = &internal_assistant_entity.Assistant{
			AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
				AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
					IdleTimeout:        &idleTimeout,
					IdleTimeoutMessage: &message,
				},
			},
		}
		r.sessionLifecycle = adapter_lifecycle.NewSessionLifecycle()
		r.sessionLifecycle.ConfigureTimeouts(context.Background(), r.GetID(), &r.assistant.AssistantPhoneDeployment.AssistantDeploymentBehavior, r.OnPacket)
		t.Cleanup(r.sessionLifecycle.CloseTimeouts)
		r.textToSpeechTransformer = noopSpeechToTextTransformer{}
		h := requestorDispatchHandler{r: r}
		oldContextID := r.GetID()
		r.messageLifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: oldContextID, Text: "done"})
		require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
			Id: oldContextID, Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "done"},
		}))
		require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
			Id: oldContextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
		}))
		require.NoError(t, r.messageLifecycle.OnPlaybackCompleted(oldContextID))
		require.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())

		h.HandleStartIdleTimeout(context.Background(), internal_type.StartIdleTimeoutPacket{ContextID: oldContextID})
		time.Sleep(10 * time.Second)
		synctest.Wait()
		var idleTimeoutExpired internal_type.IdleTimeoutExpiredPacket
		for _, packet := range drainEgressPackets(r) {
			if expired, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
				idleTimeoutExpired = expired
			}
		}
		require.NotZero(t, idleTimeoutExpired.Generation)
		h.HandleIdleTimeoutExpired(context.Background(), idleTimeoutExpired)

		newContextID := r.GetID()
		require.NotEqual(t, oldContextID, newContextID)
		assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())

		var injectMessage internal_type.InjectMessagePacket
		for _, packet := range drainEgressPackets(r) {
			switch typed := packet.(type) {
			case internal_type.InjectMessagePacket:
				injectMessage = typed
			case internal_type.StartIdleTimeoutPacket:
				t.Fatalf("idle prompt should not restart idle before assistant completion: %+v", typed)
			}
		}
		require.Equal(t, newContextID, injectMessage.ContextID)
		require.Equal(t, message, injectMessage.Text)

		h.HandleInjectMessage(context.Background(), injectMessage)
		assert.Equal(t, adapter_lifecycle.MessageStateAssistantGenerated, r.messageLifecycle.State())
		for _, packet := range drainEgressPackets(r) {
			if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
				t.Fatalf("injected idle prompt should not start idle before speech delivery: %+v", typed)
			}
		}

		h.HandleLLMResponseDone(context.Background(), internal_type.LLMResponseDonePacket{
			ContextID: newContextID,
			Text:      message,
		})
		assert.Equal(t, adapter_lifecycle.MessageStateAssistantGenerated, r.messageLifecycle.State())
		for _, packet := range drainEgressPackets(r) {
			if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
				t.Fatalf("idle prompt should not start idle at LLM done before TTS completion: %+v", typed)
			}
		}

		h.HandleTextToSpeechDone(context.Background(), internal_type.TextToSpeechDonePacket{
			ContextID: newContextID,
			Text:      message,
		})
		assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, r.messageLifecycle.State())
		for _, packet := range drainEgressPackets(r) {
			if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
				t.Fatalf("audio idle prompt should wait for playback completion before idle restart: %+v", typed)
			}
		}

		h.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: newContextID})

		for _, packet := range drainEgressPackets(r) {
			if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
				t.Fatalf("idle prompt should not emit idle timeout packet after TTS end: %+v", typed)
			}
		}
		assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, r.messageLifecycle.State())
		assert.False(t, r.messageLifecycle.CanStartIdleTimeout(newContextID))
	})
}

func TestHandleLLMResponseDone_DoesNotStartIdleTimeout(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()

	h.HandleLLMResponseDone(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: contextID,
		Text:      "done",
	})

	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			t.Fatalf("LLM done should not start idle timeout before assistant finishes: %+v", typed)
		}
	}
}

func TestHandleTextToSpeechDone_TextModeCompletesAfterDelivery(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	r.messageLifecycle = adapter_lifecycle.NewMessageLifecycle(
		adapter_lifecycle.WithContextID("ctx-active"), adapter_lifecycle.WithMode(type_enums.TextMode),
		adapter_lifecycle.WithOnPacket(func(packets ...internal_type.Packet) error { return r.OnPacket(context.Background(), packets...) }),
		adapter_lifecycle.WithSend(func(message proto.Message) error { return r.streamer.Send(message) }),
	)
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()
	require.NoError(t, r.messageLifecycle.OnGenerationStarted(contextID))
	r.messageLifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: contextID, Text: "done"})

	h.HandleTextToSpeechDone(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: contextID,
		Text:      "done",
	})

	assert.Contains(t, drainEgressPackets(r), internal_type.StartIdleTimeoutPacket{ContextID: contextID})
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())
}

func TestHandleTextToSpeechDone_AudioModeDoesNotEmitIdleTimeout(t *testing.T) {
	r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	r.textToSpeechTransformer = noopSpeechToTextTransformer{}
	h := requestorDispatchHandler{r: r}
	contextID := r.GetID()
	h.HandleLLMResponseDone(context.Background(), internal_type.LLMResponseDonePacket{ContextID: contextID, Text: "done"})

	h.HandleTextToSpeechDone(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: contextID,
		Text:      "done",
	})

	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			t.Fatalf("audio TTS done should wait for playback completion before idle timeout: %+v", typed)
		}
	}
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, r.messageLifecycle.State())

	h.HandleTextToSpeechEnd(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: contextID})

	for _, packet := range drainEgressPackets(r) {
		if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
			t.Fatalf("audio TTS end should not emit idle timeout packet: %+v", typed)
		}
	}
	assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, r.messageLifecycle.State())
	assert.False(t, r.messageLifecycle.CanStartIdleTimeout(contextID))
}

func TestHandleUnclearInputExpired_AudioModeBlocksInputUntilTextToSpeechEnd(t *testing.T) {
	r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 0.03, "Please say that again.")
	t.Cleanup(r.messageLifecycle.StopUnclearInput)
	h := requestorDispatchHandler{r: r}
	turn, err := r.messageLifecycle.OnUserTurnStarted(r.GetID(), "test", "text", "unclear input")
	require.NoError(t, err)
	contextID, err := r.messageLifecycle.OnTranscriptReceived(internal_type.SpeechToTextPacket{ContextID: turn.ContextID, Script: "unclear input", Interim: true})
	require.NoError(t, err)

	h.HandleUnclearInputExpired(context.Background(), internal_type.UnclearInputExpiredPacket{ContextID: contextID})

	newContextID := r.GetID()
	require.NotEqual(t, contextID, newContextID)
	controlPackets := drainControlPackets(r)
	require.Len(t, controlPackets, 7)

	turnChange, ok := controlPackets[1].(internal_type.TurnChangePacket)
	require.True(t, ok)
	assert.Equal(t, newContextID, turnChange.ContextID)
	assert.Equal(t, contextID, turnChange.PreviousContextID)

	audioPolicy, ok := controlPackets[4].(internal_type.DispatchPolicyPacket)
	require.True(t, ok)
	assert.Equal(t, newContextID, audioPolicy.ContextID)
	assert.Equal(t, internal_type.PacketNameUserAudioReceived, audioPolicy.Policy.Target)
	assert.Equal(t, internal_type.DispatchActionIgnore, audioPolicy.Policy.Action)

	textPolicy, ok := controlPackets[5].(internal_type.DispatchPolicyPacket)
	require.True(t, ok)
	assert.Equal(t, newContextID, textPolicy.ContextID)
	assert.Equal(t, internal_type.PacketNameUserTextReceived, textPolicy.Policy.Target)
	assert.Equal(t, internal_type.DispatchActionIgnore, textPolicy.Policy.Action)

	interruptionPolicy, ok := controlPackets[6].(internal_type.DispatchPolicyPacket)
	require.True(t, ok)
	assert.Equal(t, newContextID, interruptionPolicy.ContextID)
	assert.Equal(t, internal_type.PacketNameInterruptionDetected, interruptionPolicy.Policy.Target)
	assert.Equal(t, internal_type.DispatchActionIgnore, interruptionPolicy.Policy.Action)

	ttsInterrupt, ok := controlPackets[2].(internal_type.TextToSpeechInterruptPacket)
	require.True(t, ok)
	assert.Equal(t, contextID, ttsInterrupt.ContextID)
}
