package adapter_internal

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	adapter_router "github.com/rapidaai/api/assistant-api/internal/adapters/router"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializationFailedPacket_NonRecoverableError_NotifiesClientErrorAndDisconnection(t *testing.T) {
	streamer := &streamTestStreamer{}
	sessionContext, cancelSession := context.WithCancel(context.Background())
	t.Cleanup(cancelSession)

	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		streamer:         streamer,
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		sessionCtx:       sessionContext,
		cancelSession:    cancelSession,
		channels:         requestorChannels,
	}

	requestor.dispatch(context.Background(), internal_type.InitializationFailedPacket{
		ContextID: "ctx-init-error",
		Stage:     internal_type.InitializationStageTextToSpeech,
		Error:     errors.New("tts provider rejected credentials"),
	})

	require.Len(t, streamer.sent, 2)
	conversationError, ok := streamer.sent[0].(*protos.ConversationError)
	require.True(t, ok, "expected ConversationError, got %T", streamer.sent[0])
	assert.Contains(t, conversationError.GetMessage(), "init[tts]")

	conversationDisconnection, ok := streamer.sent[1].(*protos.ConversationDisconnection)
	require.True(t, ok, "expected ConversationDisconnection, got %T", streamer.sent[1])
	assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR, conversationDisconnection.GetType())
	assert.Equal(t, adapter_lifecycle.StateFailed, requestor.sessionLifecycle.Current())
}

func TestInitializationFailedPacket_DisconnectFinalization_CancelsSessionContext(t *testing.T) {
	streamer := &streamTestStreamer{}
	sessionContext, cancelSession := context.WithCancel(context.Background())
	t.Cleanup(cancelSession)

	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		streamer:                  streamer,
		assistantConversation:     &internal_conversation_entity.AssistantConversation{},
		messageLifecycle:          adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:          adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:             adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		sessionCtx:                sessionContext,
		cancelSession:             cancelSession,
		channels:                  requestorChannels,
		assistantAnalyseExecutors: []internal_type.AnalysisExecutor{},
	}
	requestor.assistantConversation.Id = 707

	go requestor.runBootstrapDispatcher(sessionContext)
	go requestor.runDataDispatcher(sessionContext)

	requestor.dispatch(context.Background(), internal_type.InitializationFailedPacket{
		ContextID: "ctx-init-disconnect",
		Stage:     internal_type.InitializationStageTextToSpeech,
		Error:     errors.New("tts provider rejected credentials"),
	})
	require.Equal(t, adapter_lifecycle.StateFailed, requestor.sessionLifecycle.Current())

	// Production streamers close after ConversationDisconnection; Talk then calls OnDisconnect.
	requestor.OnDisconnect(context.Background())

	require.Eventually(t, func() bool {
		return sessionContext.Err() != nil
	}, time.Second, 10*time.Millisecond)
	assert.Equal(t, adapter_lifecycle.StateDisconnected, requestor.sessionLifecycle.Current())
}

func TestInitializeTextToSpeechPacket_ConfigError_EmitsNonRecoverableInitializationFailed(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
		options:          map[string]interface{}{},
	}

	requestorDispatchHandler{r: requestor}.HandleInitializeTextToSpeech(context.Background(), internal_type.InitializeTextToSpeechPacket{
		ContextID: "ctx-tts-config-error",
		Config:    &protos.ConversationInitialization{},
	})

	select {
	case <-requestor.channels.BootstrapChannel().Ready():
		envelope := receiveEnvelope(t, requestor.channels.BootstrapChannel())
		initializationFailedPacket, ok := envelope.Pkt.(internal_type.InitializationFailedPacket)
		require.True(t, ok, "expected InitializationFailedPacket, got %T", envelope.Pkt)
		assert.Equal(t, "ctx-tts-config-error", initializationFailedPacket.ContextID)
		assert.Equal(t, internal_type.InitializationStageTextToSpeech, initializationFailedPacket.Stage)
		assert.False(t, initializationFailedPacket.IsRecoverable())
		assert.Error(t, initializationFailedPacket.Error)
	default:
		t.Fatal("expected InitializationFailedPacket in bootstrap channel")
	}
	assert.Nil(t, requestor.textToSpeechTransformer)
}

func TestInitializeBehavior_InvalidTimeoutRejectsGreeting(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options utils.Option
	}{
		{name: "idle timeout multiplication overflow", options: utils.Option{
			internal_options.ExperienceOptionIdleTimeout: uint64(math.MaxInt64/int64(time.Second)) + 1,
		}},
		{name: "maximum session signed overflow", options: utils.Option{
			internal_options.ExperienceOptionMaxSessionDuration: uint64(math.MaxUint64),
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			greeting := "Welcome!"
			requestorChannels := adapter_channel.NewRequestorChannels()
			requestor := &genericRequestor{
				source:  utils.Debugger,
				options: testCase.options,
				assistant: &internal_assistant_entity.Assistant{
					AssistantDebuggerDeployment: &internal_assistant_entity.AssistantDebuggerDeployment{
						AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{Greeting: &greeting},
					},
				},
				messageLifecycle: adapter_lifecycle.NewMessageLifecycle(adapter_lifecycle.WithContextID("invalid-timeout"), adapter_lifecycle.WithMode(type_enums.TextMode)),
				sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
				dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
				channels:         requestorChannels,
			}
			t.Cleanup(requestor.sessionLifecycle.CloseTimeouts)
			requestorDispatchHandler{r: requestor}.HandleInitializeBehavior(context.Background(), internal_type.InitializeBehaviorPacket{
				ContextID: "invalid-timeout",
			})
			require.Equal(t, 1, requestorChannels.BootstrapChannel().Len())
			failure, ok := receiveEnvelope(t, requestorChannels.BootstrapChannel()).Pkt.(internal_type.InitializationFailedPacket)
			require.True(t, ok)
			assert.Equal(t, internal_type.InitializationStageBehavior, failure.Stage)
			assert.Equal(t, "invalid-timeout", failure.ContextID)
			require.Error(t, failure.Error)
			assert.False(t, failure.IsRecoverable())
			assert.Zero(t, requestorChannels.EgressChannel().Len(), "invalid settings must not queue a greeting")
		})
	}
}

func TestInitializeBehaviorLoadsLifecycleBehaviorBeforeGreeting(t *testing.T) {
	for _, fails := range []bool{false, true} {
		name := "success preserves state"
		if fails {
			name = "failure rejects greeting"
		}
		t.Run(name, func(t *testing.T) {
			greeting := "Welcome!"
			requestor := &genericRequestor{
				source: utils.Debugger,
				assistant: &internal_assistant_entity.Assistant{
					AssistantDebuggerDeployment: &internal_assistant_entity.AssistantDebuggerDeployment{
						AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{Greeting: &greeting},
					},
				},
				channels: adapter_channel.NewRequestorChannels(),
			}
			failure := errors.New("lazy behavior load failed")
			calls := 0
			requestor.messageLifecycle = adapter_lifecycle.NewMessageLifecycle(
				adapter_lifecycle.WithContextID("ctx-lazy-behavior"),
				adapter_lifecycle.WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
					calls++
					require.Empty(t, drainEgressPackets(requestor), "greeting must wait for lifecycle initialization")
					if fails {
						return nil, failure
					}
					return requestor.deploymentBehavior()
				}),
			)
			t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
			original := requestor.messageLifecycle
			require.NoError(t, original.OnGenerationStarted(requestor.GetID()))
			require.Zero(t, calls, "behavior must not load during construction")

			requestorDispatchHandler{r: requestor}.HandleInitializeBehavior(context.Background(), internal_type.InitializeBehaviorPacket{
				ContextID: requestor.GetID(),
			})

			require.Equal(t, 1, calls)
			require.Same(t, original, requestor.messageLifecycle)
			require.Equal(t, "ctx-lazy-behavior", requestor.GetID())
			require.Equal(t, adapter_lifecycle.MessageStateAssistantGenerating, original.State())
			if fails {
				require.Equal(t, 1, requestor.channels.BootstrapChannel().Len())
				packet, ok := receiveEnvelope(t, requestor.channels.BootstrapChannel()).Pkt.(internal_type.InitializationFailedPacket)
				require.True(t, ok)
				require.Equal(t, internal_type.InitializationStageBehavior, packet.Stage)
				require.Equal(t, requestor.GetID(), packet.ContextID)
				require.ErrorIs(t, packet.Error, failure)
				require.Empty(t, drainEgressPackets(requestor))
				return
			}
			require.Zero(t, requestor.channels.BootstrapChannel().Len())
			require.Equal(t, []internal_type.Packet{internal_type.InjectMessagePacket{ContextID: requestor.GetID(), Text: greeting}}, drainEgressPackets(requestor))
		})
	}
}

func TestInitializeBehavior_GreetingInterruptibleOption_ControlsAudioBlock(t *testing.T) {
	greeting := "Welcome!"
	interruptibleGreeting := true
	nonInterruptibleGreeting := false

	testCases := []struct {
		name                  string
		greetingInterruptible *bool
		expectAudioBlocked    bool
	}{
		{
			name:                  "without interruptible option audio is not blocked",
			greetingInterruptible: nil,
			expectAudioBlocked:    false,
		},
		{
			name:                  "with interruptible greeting audio is not blocked",
			greetingInterruptible: &interruptibleGreeting,
			expectAudioBlocked:    false,
		},
		{
			name:                  "with non interruptible greeting audio is blocked",
			greetingInterruptible: &nonInterruptibleGreeting,
			expectAudioBlocked:    true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			requestorChannels := adapter_channel.NewRequestorChannels()
			requestor := &genericRequestor{
				source:                  utils.Debugger,
				speechToTextTransformer: noopSpeechToTextTransformer{},
				endOfSpeechExecutor:     &recordingEOSExecutor{},
				assistant: &internal_assistant_entity.Assistant{
					AssistantDebuggerDeployment: &internal_assistant_entity.AssistantDebuggerDeployment{
						AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
							Greeting:              &greeting,
							GreetingInterruptible: testCase.greetingInterruptible,
						},
					},
				},
				messageLifecycle: adapter_lifecycle.NewMessageLifecycle(adapter_lifecycle.WithContextID("ctx-greeting-init"), adapter_lifecycle.WithMode(type_enums.AudioMode)),
				sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
				dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
				channels:         requestorChannels,
			}

			requestorDispatchHandler{r: requestor}.HandleInitializeBehavior(context.Background(), internal_type.InitializeBehaviorPacket{
				ContextID: "ctx-greeting-init",
				Config:    &protos.ConversationInitialization{StreamMode: protos.StreamMode_STREAM_MODE_AUDIO},
			})

			for requestor.channels.ControlChannel().Len() > 0 {
				requestor.dispatch(context.Background(), receiveEnvelope(t, requestor.channels.ControlChannel()).Pkt)
			}
			requestor.dispatch(context.Background(), internal_type.UserAudioReceivedPacket{
				ContextID: "ctx-greeting-init",
				Audio:     []byte("audio"),
			})

			if testCase.expectAudioBlocked {
				assert.Zero(t, requestor.channels.IngressChannel().Len())
			} else {
				require.Equal(t, 2, requestor.channels.IngressChannel().Len())
				assert.Equal(t, internal_type.PacketNameSpeechToTextAudio, receiveEnvelope(t, requestor.channels.IngressChannel()).Pkt.PacketName())
				assert.Equal(t, internal_type.PacketNameEndOfSpeechAudio, receiveEnvelope(t, requestor.channels.IngressChannel()).Pkt.PacketName())
			}

			select {
			case <-requestor.channels.EgressChannel().Ready():
				envelope := receiveEnvelope(t, requestor.channels.EgressChannel())
				injectMessagePacket, ok := envelope.Pkt.(internal_type.InjectMessagePacket)
				require.True(t, ok, "expected InjectMessagePacket, got %T", envelope.Pkt)
				assert.Equal(t, greeting, injectMessagePacket.Text)
			default:
				t.Fatal("expected greeting inject message")
			}
		})
	}
}

func TestInitializeBehavior_GreetingDoesNotStartIdleTimeoutBeforeCompletion(t *testing.T) {
	greeting := "Welcome!"
	idleTimeout := uint64(10)
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		source: utils.Debugger,
		assistant: &internal_assistant_entity.Assistant{
			AssistantDebuggerDeployment: &internal_assistant_entity.AssistantDebuggerDeployment{
				AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
					Greeting:    &greeting,
					IdleTimeout: &idleTimeout,
				},
			},
		},
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(adapter_lifecycle.WithContextID("ctx-greeting-idle"), adapter_lifecycle.WithMode(type_enums.TextMode)),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}

	requestorDispatchHandler{r: requestor}.HandleInitializeBehavior(context.Background(), internal_type.InitializeBehaviorPacket{
		ContextID: "ctx-greeting-idle",
		Config:    &protos.ConversationInitialization{StreamMode: protos.StreamMode_STREAM_MODE_TEXT},
	})

	var injectMessage internal_type.InjectMessagePacket
	for requestor.channels.EgressChannel().Len() > 0 {
		packet := receiveEnvelope(t, requestor.channels.EgressChannel()).Pkt
		switch typed := packet.(type) {
		case internal_type.InjectMessagePacket:
			injectMessage = typed
		case internal_type.StartIdleTimeoutPacket:
			t.Fatalf("greeting should not start idle timeout before assistant completion: %+v", typed)
		}
	}
	assert.Equal(t, greeting, injectMessage.Text)
}

func TestInitializeBehavior_DoesNotEmitIdleTimeoutWhenNoGreetingIsInjected(t *testing.T) {
	idleTimeout := uint64(10)
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		source: utils.Debugger,
		assistant: &internal_assistant_entity.Assistant{
			AssistantDebuggerDeployment: &internal_assistant_entity.AssistantDebuggerDeployment{
				AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
					IdleTimeout: &idleTimeout,
				},
			},
		},
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(adapter_lifecycle.WithContextID("ctx-no-greeting-idle"), adapter_lifecycle.WithMode(type_enums.TextMode)),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}

	requestorDispatchHandler{r: requestor}.HandleInitializeBehavior(context.Background(), internal_type.InitializeBehaviorPacket{
		ContextID: "ctx-no-greeting-idle",
		Config:    &protos.ConversationInitialization{StreamMode: protos.StreamMode_STREAM_MODE_TEXT},
	})

	for requestor.channels.EgressChannel().Len() > 0 {
		if typed, ok := receiveEnvelope(t, requestor.channels.EgressChannel()).Pkt.(internal_type.StartIdleTimeoutPacket); ok {
			t.Fatalf("initialization should not emit idle timeout packet: %+v", typed)
		}
	}
}

func TestInitializeBehavior_NonInterruptibleGreeting_BlocksAudioAndAcceptsAfterTextToSpeechEnd(t *testing.T) {
	greeting := "Welcome!"
	nonInterruptibleGreeting := false
	streamer := &streamTestStreamer{}
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		source:                  utils.Debugger,
		streamer:                streamer,
		speechToTextTransformer: noopSpeechToTextTransformer{},
		endOfSpeechExecutor:     &recordingEOSExecutor{},
		assistant: &internal_assistant_entity.Assistant{
			AssistantDebuggerDeployment: &internal_assistant_entity.AssistantDebuggerDeployment{
				AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
					Greeting:              &greeting,
					GreetingInterruptible: &nonInterruptibleGreeting,
				},
			},
		},
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(adapter_lifecycle.WithContextID("ctx-greeting-audio"), adapter_lifecycle.WithMode(type_enums.AudioMode)),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}

	requestorDispatchHandler{r: requestor}.HandleInitializeBehavior(context.Background(), internal_type.InitializeBehaviorPacket{
		ContextID: "ctx-greeting-audio",
		Config:    &protos.ConversationInitialization{StreamMode: protos.StreamMode_STREAM_MODE_AUDIO},
	})
	for requestor.channels.ControlChannel().Len() > 0 {
		requestor.dispatch(context.Background(), receiveEnvelope(t, requestor.channels.ControlChannel()).Pkt)
	}
	for requestor.channels.EgressChannel().Len() > 0 {
		_ = receiveEnvelope(t, requestor.channels.EgressChannel())
	}

	requestor.dispatch(context.Background(), internal_type.UserAudioReceivedPacket{
		ContextID: "ctx-greeting-audio",
		Audio:     []byte("audio"),
	})
	requestor.dispatch(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-greeting-audio",
		Source:    internal_type.InterruptionSourceWord,
	})
	assert.Zero(t, requestor.channels.IngressChannel().Len())
	assert.Zero(t, requestor.channels.ControlChannel().Len())
	assert.Zero(t, requestor.channels.EgressChannel().Len())

	requestor.dispatch(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: "ctx-greeting-audio"})
	for requestor.channels.ControlChannel().Len() > 0 {
		requestor.dispatch(context.Background(), receiveEnvelope(t, requestor.channels.ControlChannel()).Pkt)
	}
	requestor.dispatch(context.Background(), internal_type.UserAudioReceivedPacket{
		ContextID: "ctx-greeting-audio",
		Audio:     []byte("audio-after-greeting"),
	})

	require.Equal(t, 2, requestor.channels.IngressChannel().Len())
	assert.Equal(t, internal_type.PacketNameSpeechToTextAudio, receiveEnvelope(t, requestor.channels.IngressChannel()).Pkt.PacketName())
	assert.Equal(t, internal_type.PacketNameEndOfSpeechAudio, receiveEnvelope(t, requestor.channels.IngressChannel()).Pkt.PacketName())
}

func TestInitializeBehavior_NonInterruptibleGreeting_TextInputDoesNotKeepAudioBlockedAfterTextToSpeechEnd(t *testing.T) {
	greeting := "Welcome!"
	nonInterruptibleGreeting := false
	streamer := &streamTestStreamer{}
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		source:                  utils.Debugger,
		streamer:                streamer,
		speechToTextTransformer: noopSpeechToTextTransformer{},
		endOfSpeechExecutor:     &recordingEOSExecutor{},
		assistant: &internal_assistant_entity.Assistant{
			AssistantDebuggerDeployment: &internal_assistant_entity.AssistantDebuggerDeployment{
				AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
					Greeting:              &greeting,
					GreetingInterruptible: &nonInterruptibleGreeting,
				},
			},
		},
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(adapter_lifecycle.WithContextID("ctx-greeting-text"), adapter_lifecycle.WithMode(type_enums.AudioMode)),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}

	requestorDispatchHandler{r: requestor}.HandleInitializeBehavior(context.Background(), internal_type.InitializeBehaviorPacket{
		ContextID: "ctx-greeting-text",
		Config:    &protos.ConversationInitialization{StreamMode: protos.StreamMode_STREAM_MODE_AUDIO},
	})
	for requestor.channels.ControlChannel().Len() > 0 {
		requestor.dispatch(context.Background(), receiveEnvelope(t, requestor.channels.ControlChannel()).Pkt)
	}
	for requestor.channels.EgressChannel().Len() > 0 {
		_ = receiveEnvelope(t, requestor.channels.EgressChannel())
	}

	requestor.dispatch(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-greeting-text",
		Text:      "interrupt with text",
	})
	requestor.channels.FlushAll()

	requestor.dispatch(context.Background(), internal_type.TextToSpeechEndPacket{ContextID: "ctx-greeting-text"})
	for requestor.channels.ControlChannel().Len() > 0 {
		requestor.dispatch(context.Background(), receiveEnvelope(t, requestor.channels.ControlChannel()).Pkt)
	}
	requestor.dispatch(context.Background(), internal_type.UserAudioReceivedPacket{
		ContextID: requestor.GetID(),
		Audio:     []byte("audio-after-greeting"),
	})

	require.Equal(t, 2, requestor.channels.IngressChannel().Len())
	assert.Equal(t, internal_type.PacketNameSpeechToTextAudio, receiveEnvelope(t, requestor.channels.IngressChannel()).Pkt.PacketName())
	assert.Equal(t, internal_type.PacketNameEndOfSpeechAudio, receiveEnvelope(t, requestor.channels.IngressChannel()).Pkt.PacketName())
}
