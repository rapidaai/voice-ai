package adapter_internal

import (
	"context"
	"errors"
	"testing"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	adapter_router "github.com/rapidaai/api/assistant-api/internal/adapters/router"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingObservabilityRecorder struct {
	scopes  []observability.Scope
	records []observability.Record
	order   *[]string
}

func (r *recordingObservabilityRecorder) Record(_ context.Context, scope observability.Scope, records ...observability.Record) error {
	for _, record := range records {
		if r.order != nil {
			*r.order = append(*r.order, "record-webhook")
		}
		r.scopes = append(r.scopes, scope)
		r.records = append(r.records, record)
	}
	return nil
}

func (r *recordingObservabilityRecorder) AddCollectors(...observability.Collector) error {
	return nil
}

func (r *recordingObservabilityRecorder) Close(context.Context) error {
	return nil
}

type recordingAnalysisExecutor struct {
	order        *[]string
	execute      func(internal_type.AnalysisInput) *protos.Metadata
	executeCalls int
	closeCalls   int
}

func (e *recordingAnalysisExecutor) Name() string {
	return "recording-analysis"
}

func (e *recordingAnalysisExecutor) Options() utils.Option {
	return nil
}

func (e *recordingAnalysisExecutor) Arguments() (map[string]string, error) {
	return nil, nil
}

func (e *recordingAnalysisExecutor) Execute(_ context.Context, input internal_type.AnalysisInput) (*internal_type.AnalysisOutput, error) {
	if e.order != nil {
		*e.order = append(*e.order, "analysis-execute")
	}
	e.executeCalls++
	if e.execute != nil {
		return &internal_type.AnalysisOutput{Metadata: e.execute(input)}, nil
	}
	return &internal_type.AnalysisOutput{}, nil
}

func (e *recordingAnalysisExecutor) Close(context.Context) error {
	if e.order != nil {
		*e.order = append(*e.order, "analysis-close")
	}
	e.closeCalls++
	return nil
}

type recordingAuthenticationExecutor struct {
	arguments    map[string]string
	result       *internal_type.AuthenticationOutput
	err          error
	input        internal_type.AuthenticationInput
	executeCalls int
	closeCalls   int
}

func (e *recordingAuthenticationExecutor) Name() string {
	return "recording-authentication"
}

func (e *recordingAuthenticationExecutor) Options() utils.Option {
	return nil
}

func (e *recordingAuthenticationExecutor) Arguments() (map[string]string, error) {
	return e.arguments, nil
}

func (e *recordingAuthenticationExecutor) Execute(_ context.Context, input internal_type.AuthenticationInput) (*internal_type.AuthenticationOutput, error) {
	e.input = input
	e.executeCalls++
	return e.result, e.err
}

func (e *recordingAuthenticationExecutor) Close(context.Context) error {
	e.closeCalls++
	return nil
}

func runFinalizeDataFlow(requestor *genericRequestor, contextID string) []internal_type.PacketName {
	handler := requestorDispatchHandler{r: requestor}
	handler.HandleFinalizeAuthentication(context.Background(), internal_type.FinalizeAuthenticationPacket{ContextID: contextID})

	packetNames := make([]internal_type.PacketName, 0)
	for len(requestor.channels.DataChannel()) > 0 {
		envelope := <-requestor.channels.DataChannel()
		packetNames = append(packetNames, envelope.Pkt.PacketName())
		switch packet := envelope.Pkt.(type) {
		case internal_type.FinalizeConversationRecordingExecutorPacket:
			handler.HandleFinalizeConversationRecordingExecutor(envelope.Ctx, packet)
		case internal_type.FinalizeSessionRuntimePacket:
			handler.HandleFinalizeSessionRuntime(envelope.Ctx, packet)
		case internal_type.FinalizeArtifactPushExecutorPacket:
			handler.HandleFinalizeArtifactPushExecutor(envelope.Ctx, packet)
		case internal_type.ExecuteAnalysisPacket:
			handler.HandleExecuteAnalysis(envelope.Ctx, packet)
		case internal_type.FinalizeConversationPacket:
			handler.HandleFinalizeConversation(envelope.Ctx, packet)
		case internal_type.FinalizeAnalysisExecutorPacket:
			handler.HandleFinalizeAnalysisExecutor(envelope.Ctx, packet)
		case internal_type.FinalizeAssistantPacket:
			handler.HandleFinalizeAssistant(envelope.Ctx, packet)
		case internal_type.FinalizationCompletedPacket:
			handler.HandleFinalizationCompleted(envelope.Ctx, packet)
		}
	}
	return packetNames
}

func TestInitializationPackets_RouteToBootstrapChannel(t *testing.T) {
	conversationInitialization := &protos.ConversationInitialization{}
	initializationError := errors.New("initialization failed")

	initializationPackets := []internal_type.Packet{
		internal_type.InitializeAssistantPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeConversationPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeSessionRuntimePacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeConversationRecordingExecutorPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeArtifactPushExecutorPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeAnalysisExecutorPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeAuthenticationPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.ExecuteAuthenticationPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.SessionAuthenticationSucceededPacket{ContextID: "ctx", Initialization: conversationInitialization},
		internal_type.SessionAuthenticationFailedPacket{ContextID: "ctx", Initialization: conversationInitialization, Error: initializationError},
		internal_type.InitializeSpeechToTextPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeTextToSpeechPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeAssistantExecutorPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeVoiceActivityDetectionPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeEndOfSpeechPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeDenoisePacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeBehaviorPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializationCompletedPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializationFailedPacket{ContextID: "ctx", Stage: internal_type.InitializationStageService, Error: initializationError},
		internal_type.InitializeInboundDispatcherPacket{ContextID: "ctx"},
	}

	for _, initializationPacket := range initializationPackets {
		t.Run(string(initializationPacket.PacketName()), func(t *testing.T) {
			requestorChannels := adapter_channel.NewRequestorChannels()
			requestor := &genericRequestor{channels: requestorChannels}

			err := requestor.OnPacket(context.Background(), initializationPacket)
			require.NoError(t, err)

			select {
			case envelope := <-requestor.channels.BootstrapChannel():
				assert.Equal(t, initializationPacket.PacketName(), envelope.Pkt.PacketName())
			default:
				t.Fatalf("expected %s in bootstrap channel", initializationPacket.PacketName())
			}

			assert.Empty(t, requestor.channels.ControlChannel())
			assert.Empty(t, requestor.channels.IngressChannel())
			assert.Empty(t, requestor.channels.EgressChannel())
			assert.Empty(t, requestor.channels.DataChannel())
			assert.Empty(t, requestor.channels.BackgroundChannel())
		})
	}
}

func TestHandleInitializeAuthentication_QueuesAuthenticationExecution(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	initialization := &protos.ConversationInitialization{
		StreamMode: protos.StreamMode_STREAM_MODE_TEXT,
	}
	requestor := &genericRequestor{
		assistant:        &internal_assistant_entity.Assistant{},
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}

	requestorDispatchHandler{r: requestor}.HandleInitializeAuthentication(context.Background(), internal_type.InitializeAuthenticationPacket{
		ContextID: "ctx-auth",
		Config:    initialization,
	})

	select {
	case envelope := <-requestor.channels.BootstrapChannel():
		packet, ok := envelope.Pkt.(internal_type.ExecuteAuthenticationPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-auth", packet.ContextID)
		assert.Same(t, initialization, packet.Config)
	default:
		t.Fatalf("expected execute authentication packet")
	}
}

func TestHandleExecuteAuthentication_ExecutesAuthenticationSynchronously(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	initialization := &protos.ConversationInitialization{
		StreamMode: protos.StreamMode_STREAM_MODE_TEXT,
	}
	authentication := &recordingAuthenticationExecutor{
		arguments: map[string]string{
			"token": "abc",
		},
		result: &internal_type.AuthenticationOutput{
			Authenticated: true,
			Arguments: map[string]interface{}{
				"user_id": "user-1",
			},
			Metadata: map[string]interface{}{
				"tier": "gold",
			},
			Options: map[string]interface{}{
				"language": "en",
			},
		},
	}
	requestor := &genericRequestor{
		authenticationExecutor: authentication,
		messageLifecycle:       adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:       adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:          adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:               requestorChannels,
	}

	requestorDispatchHandler{r: requestor}.HandleExecuteAuthentication(context.Background(), internal_type.ExecuteAuthenticationPacket{
		ContextID: "ctx-auth",
		Config:    initialization,
	})

	assert.Equal(t, 1, authentication.executeCalls)
	assert.Equal(t, "ctx-auth", authentication.input.ContextID)
	assert.NotNil(t, authentication.input.Arguments)
	assert.Same(t, initialization, authentication.input.Initialization)
	select {
	case envelope := <-requestor.channels.BackgroundChannel():
		packet, ok := envelope.Pkt.(internal_type.ObservabilityEventRecordPacket)
		require.True(t, ok)
		assert.Equal(t, observability.ConversationAuthenticationStarted, packet.Record.Event)
	default:
		t.Fatalf("expected authentication started event")
	}
	select {
	case envelope := <-requestor.channels.BootstrapChannel():
		packet, ok := envelope.Pkt.(internal_type.SessionAuthenticationSucceededPacket)
		require.True(t, ok)
		assert.True(t, packet.Authenticated)
		assert.Equal(t, "user-1", packet.Arguments["user_id"])
		assert.Equal(t, "gold", packet.Metadata["tier"])
		assert.Equal(t, "en", packet.Options["language"])
		assert.Same(t, initialization, packet.Initialization)
	default:
		t.Fatalf("expected authentication succeeded packet")
	}
}

func TestHandleExecuteAuthentication_ExecutionErrorEnqueuesFailedPacket(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	authError := errors.New("authentication: endpoint returned status 401")
	authentication := &recordingAuthenticationExecutor{
		arguments: map[string]string{},
		err:       authError,
	}
	requestor := &genericRequestor{
		authenticationExecutor: authentication,
		messageLifecycle:       adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:       adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:          adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:               requestorChannels,
	}

	requestorDispatchHandler{r: requestor}.HandleExecuteAuthentication(context.Background(), internal_type.ExecuteAuthenticationPacket{
		ContextID: "ctx-auth-failed",
		Config:    &protos.ConversationInitialization{},
	})

	assert.Equal(t, 1, authentication.executeCalls)
	select {
	case envelope := <-requestor.channels.BootstrapChannel():
		packet, ok := envelope.Pkt.(internal_type.SessionAuthenticationFailedPacket)
		require.True(t, ok)
		assert.ErrorIs(t, packet.Error, authError)
	default:
		t.Fatalf("expected authentication failed packet")
	}
}

func TestHandleFinalizeAuthentication_ClosesExecutorAndQueuesRecordingFinalization(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	authentication := &recordingAuthenticationExecutor{}
	requestor := &genericRequestor{
		authenticationExecutor: authentication,
		messageLifecycle:       adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:       adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:          adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:               requestorChannels,
	}

	requestorDispatchHandler{r: requestor}.HandleFinalizeAuthentication(context.Background(), internal_type.FinalizeAuthenticationPacket{
		ContextID: "ctx-auth-finalize",
	})

	assert.Equal(t, 1, authentication.closeCalls)
	assert.Nil(t, requestor.authenticationExecutor)
	select {
	case envelope := <-requestor.channels.DataChannel():
		packet, ok := envelope.Pkt.(internal_type.FinalizeConversationRecordingExecutorPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-auth-finalize", packet.ContextID)
	default:
		t.Fatalf("expected finalize conversation recording executor packet")
	}
}

func TestHandleSessionAuthenticationSucceeded_TextMode_EnqueuesTextInitializationPackets(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		source:           utils.Debugger,
		assistant:        &internal_assistant_entity.Assistant{},
		args:             map[string]interface{}{},
		metadata:         map[string]interface{}{},
		options:          map[string]interface{}{},
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}
	requestor.assistant.Id = 101
	requestor.assistant.AssistantProviderId = 202
	requestor.assistantConversation = &internal_conversation_entity.AssistantConversation{}
	requestor.assistantConversation.Id = 303

	requestorDispatchHandler{r: requestor}.HandleSessionAuthenticationSucceeded(context.Background(), internal_type.SessionAuthenticationSucceededPacket{
		ContextID: "ctx-text-init",
		Initialization: &protos.ConversationInitialization{
			StreamMode: protos.StreamMode_STREAM_MODE_TEXT,
		},
	})

	var packetNames []internal_type.PacketName
	for len(requestor.channels.BootstrapChannel()) > 0 {
		packetNames = append(packetNames, (<-requestor.channels.BootstrapChannel()).Pkt.PacketName())
	}

	assert.Equal(t, []internal_type.PacketName{
		internal_type.PacketNameInitializeAssistantExecutor,
		internal_type.PacketNameInitializeBehavior,
		internal_type.PacketNameInitializationCompleted,
	}, packetNames)
	assert.Equal(t, type_enums.TextMode, requestor.GetMode())
}

func TestHandleSessionAuthenticationSucceeded_AudioMode_EnqueuesAudioInitializationPackets(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		source:           utils.Debugger,
		assistant:        &internal_assistant_entity.Assistant{},
		args:             map[string]interface{}{},
		metadata:         map[string]interface{}{},
		options:          map[string]interface{}{},
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}
	requestor.assistant.Id = 101
	requestor.assistant.AssistantProviderId = 202
	requestor.assistantConversation = &internal_conversation_entity.AssistantConversation{}
	requestor.assistantConversation.Id = 303

	requestorDispatchHandler{r: requestor}.HandleSessionAuthenticationSucceeded(context.Background(), internal_type.SessionAuthenticationSucceededPacket{
		ContextID: "ctx-audio-init",
		Initialization: &protos.ConversationInitialization{
			StreamMode: protos.StreamMode_STREAM_MODE_AUDIO,
		},
	})

	var packetNames []internal_type.PacketName
	for len(requestor.channels.BootstrapChannel()) > 0 {
		packetNames = append(packetNames, (<-requestor.channels.BootstrapChannel()).Pkt.PacketName())
	}

	assert.Equal(t, []internal_type.PacketName{
		internal_type.PacketNameInitializeSpeechToText,
		internal_type.PacketNameInitializeTextToSpeech,
		internal_type.PacketNameInitializeAssistantExecutor,
		internal_type.PacketNameInitializeVoiceActivityDetection,
		internal_type.PacketNameInitializeEndOfSpeech,
		internal_type.PacketNameInitializeDenoise,
		internal_type.PacketNameInitializeBehavior,
		internal_type.PacketNameInitializationCompleted,
	}, packetNames)
	assert.Equal(t, type_enums.AudioMode, requestor.GetMode())
}

func TestHandleInitializationCompleted_EmitsConversationWebhookRecord(t *testing.T) {
	testCases := []struct {
		name                    string
		assistantConversationID uint64
		expectedEvent           observability.EventName
		expectedDataKey         string
		expectedDataValue       interface{}
	}{
		{
			name:              "begin",
			expectedEvent:     observability.ConversationBegin,
			expectedDataKey:   "is_new",
			expectedDataValue: "true",
		},
		{
			name:                    "resume",
			assistantConversationID: 303,
			expectedEvent:           observability.ConversationResume,
			expectedDataKey:         "message_count",
			expectedDataValue:       "0",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			requestorChannels := adapter_channel.NewRequestorChannels()
			messageLifecycle := adapter_lifecycle.NewMessageLifecycleWithContext("ctx-init-webhook", type_enums.TextMode)
			requestor := &genericRequestor{
				source:           utils.Debugger,
				streamer:         &streamTestStreamer{},
				assistant:        &internal_assistant_entity.Assistant{},
				args:             map[string]interface{}{},
				metadata:         map[string]interface{}{},
				options:          map[string]interface{}{},
				messageLifecycle: messageLifecycle,
				sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
				dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
				channels:         requestorChannels,
			}
			requestor.assistant.Id = 101
			requestor.assistant.AssistantProviderId = 202
			requestor.assistantConversation = &internal_conversation_entity.AssistantConversation{}
			requestor.assistantConversation.Id = 303

			requestorDispatchHandler{r: requestor}.HandleInitializationCompleted(context.Background(), internal_type.InitializationCompletedPacket{
				ContextID: "ctx-init-webhook",
				Config: &protos.ConversationInitialization{
					AssistantConversationId: testCase.assistantConversationID,
				},
			})

			var webhookPacket internal_type.ObservabilityWebhookRecordPacket
			found := false
			for len(requestor.channels.BackgroundChannel()) > 0 {
				envelope := <-requestor.channels.BackgroundChannel()
				packet, ok := envelope.Pkt.(internal_type.ObservabilityWebhookRecordPacket)
				if !ok {
					continue
				}
				webhookPacket = packet
				found = true
			}

			require.True(t, found, "expected ObservabilityWebhookRecordPacket")
			assert.Equal(t, "ctx-init-webhook", webhookPacket.ContextID)
			assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, webhookPacket.Scope)
			assert.Equal(t, testCase.expectedEvent, webhookPacket.Record.Event)
			assert.Equal(t, testCase.expectedDataValue, webhookPacket.Record.Payload[testCase.expectedDataKey])
		})
	}
}

func TestHandleFinalizeSessionRuntime_QueuesConversationFinalization(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		assistant:                 &internal_assistant_entity.Assistant{},
		assistantConversation:     &internal_conversation_entity.AssistantConversation{},
		messageLifecycle:          adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:          adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateDisconnecting),
		dispatchRoute:             adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:                  requestorChannels,
		assistantAnalyseExecutors: []internal_type.AnalysisExecutor{},
		metadata:                  map[string]interface{}{},
		metrics:                   map[string]*protos.Metric{},
	}
	requestor.assistant.Id = 101
	requestor.assistantConversation.Id = 303

	requestorDispatchHandler{r: requestor}.HandleFinalizeSessionRuntime(context.Background(), internal_type.FinalizeSessionRuntimePacket{
		ContextID: "ctx-final-webhook",
	})

	var packetNames []internal_type.PacketName
	for len(requestor.channels.DataChannel()) > 0 {
		packetNames = append(packetNames, (<-requestor.channels.DataChannel()).Pkt.PacketName())
	}

	assert.Equal(t, []internal_type.PacketName{
		internal_type.PacketNameFinalizeArtifactPushExecutor,
	}, packetNames)
	assert.Empty(t, requestor.channels.BackgroundChannel())
}

func TestHandleFinalizeConversation_RecordsCompletedWebhookBeforeClosingAnalysis(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	order := []string{}
	recorder := &recordingObservabilityRecorder{order: &order}
	analysis := &recordingAnalysisExecutor{order: &order}
	requestor := &genericRequestor{
		assistant:             &internal_assistant_entity.Assistant{},
		assistantConversation: &internal_conversation_entity.AssistantConversation{},
		observabilityRecorder: recorder,
		messageLifecycle:      adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:      adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateDisconnecting),
		dispatchRoute:         adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:              requestorChannels,
		assistantAnalyseExecutors: []internal_type.AnalysisExecutor{
			analysis,
		},
		histories: []internal_type.MessagePacket{
			internal_type.MessageCreatePacket{
				ContextID:   "msg-user-1",
				MessageRole: "user",
				Text:        "hello",
			},
			internal_type.MessageCreatePacket{
				ContextID:   "msg-assistant-1",
				MessageRole: "assistant",
				Text:        "hi",
			},
		},
		metadata: map[string]interface{}{
			"customer_id": "customer-1",
		},
		metrics: map[string]*protos.Metric{
			"turn_count": {
				Name:        "turn_count",
				Value:       "2",
				Description: "Number of conversation turns",
			},
		},
	}
	requestor.assistant.Id = 101
	requestor.assistantConversation.Id = 303

	handler := requestorDispatchHandler{r: requestor}
	handler.HandleExecuteAnalysis(context.Background(), internal_type.ExecuteAnalysisPacket{
		ContextID: "ctx-final-webhook",
	})

	select {
	case envelope := <-requestor.channels.DataChannel():
		packet, ok := envelope.Pkt.(internal_type.FinalizeConversationPacket)
		require.True(t, ok)
		handler.HandleFinalizeConversation(envelope.Ctx, packet)
	default:
		t.Fatalf("expected FinalizeConversationPacket")
	}
	select {
	case envelope := <-requestor.channels.DataChannel():
		packet, ok := envelope.Pkt.(internal_type.FinalizeAnalysisExecutorPacket)
		require.True(t, ok)
		handler.HandleFinalizeAnalysisExecutor(envelope.Ctx, packet)
	default:
		t.Fatalf("expected FinalizeAnalysisExecutorPacket")
	}

	require.Equal(t, []string{"analysis-execute", "record-webhook", "analysis-close"}, order)
	assert.Equal(t, 1, analysis.executeCalls)
	assert.Equal(t, 1, analysis.closeCalls)
	require.Len(t, recorder.records, 1)
	scope, ok := recorder.scopes[0].(observability.ConversationScope)
	require.True(t, ok)
	assert.Equal(t, uint64(101), scope.AssistantID)
	assert.Equal(t, uint64(303), scope.ConversationID)
	webhookRecord, ok := recorder.records[0].(observability.RecordWebhook)
	require.True(t, ok)
	assert.Equal(t, observability.ConversationCompleted, webhookRecord.Event)
	assert.Equal(t, "conversation_completed", webhookRecord.Payload["reason"])
	assert.Equal(t, "completed", webhookRecord.Payload["status"])
	messagesPayload, ok := webhookRecord.Payload["messages"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, messagesPayload, 2)
	assert.Equal(t, "msg-user-1", messagesPayload[0]["id"])
	assert.Equal(t, "user", messagesPayload[0]["role"])
	assert.Equal(t, "hello", messagesPayload[0]["content"])
	metadataPayload, ok := webhookRecord.Payload["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "customer-1", metadataPayload["customer_id"])
	metricsPayload, ok := webhookRecord.Payload["metrics"].([]map[string]interface{})
	require.True(t, ok)
	metricValues := map[string]string{}
	for _, metric := range metricsPayload {
		name, _ := metric["name"].(string)
		value, _ := metric["value"].(string)
		metricValues[name] = value
	}
	assert.Equal(t, "2", metricValues["turn_count"])
	assert.Equal(t, type_enums.CONVERSATION_COMPLETE.String(), metricValues[type_enums.CONVERSATION_STATUS.String()])
	select {
	case envelope := <-requestor.channels.DataChannel():
		assert.Equal(t, internal_type.PacketNameFinalizeAssistant, envelope.Pkt.PacketName())
	default:
		t.Fatalf("expected FinalizeAssistantPacket")
	}
}

func TestHandleFinalizeConversation_RecordsCompletedWebhookWithoutAnalysis(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	recorder := &recordingObservabilityRecorder{}
	requestor := &genericRequestor{
		assistant:                 &internal_assistant_entity.Assistant{},
		assistantConversation:     &internal_conversation_entity.AssistantConversation{},
		observabilityRecorder:     recorder,
		messageLifecycle:          adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:          adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateDisconnecting),
		dispatchRoute:             adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:                  requestorChannels,
		assistantAnalyseExecutors: []internal_type.AnalysisExecutor{},
		metadata:                  map[string]interface{}{},
		metrics:                   map[string]*protos.Metric{},
	}
	requestor.assistant.Id = 101
	requestor.assistantConversation.Id = 303

	requestorDispatchHandler{r: requestor}.HandleFinalizeConversation(context.Background(), internal_type.FinalizeConversationPacket{
		ContextID: "ctx-final-webhook-no-analysis",
	})

	require.Len(t, recorder.records, 1)
	webhookRecord, ok := recorder.records[0].(observability.RecordWebhook)
	require.True(t, ok)
	assert.Equal(t, observability.ConversationCompleted, webhookRecord.Event)
	assert.Equal(t, "completed", webhookRecord.Payload["status"])
	select {
	case envelope := <-requestor.channels.DataChannel():
		assert.Equal(t, internal_type.PacketNameFinalizeAnalysisExecutor, envelope.Pkt.PacketName())
	default:
		t.Fatalf("expected FinalizeAnalysisExecutorPacket")
	}
}

func TestFinalizeFlow_WithAnalysis_RunsAnalysisThenWebhookThenEnds(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	sessionContext, cancelSession := context.WithCancel(context.Background())
	t.Cleanup(cancelSession)
	order := []string{}
	recorder := &recordingObservabilityRecorder{order: &order}
	requestor := &genericRequestor{
		assistant:             &internal_assistant_entity.Assistant{},
		assistantConversation: &internal_conversation_entity.AssistantConversation{},
		observabilityRecorder: recorder,
		messageLifecycle:      adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:      adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateDisconnecting),
		dispatchRoute:         adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:              requestorChannels,
		sessionCtx:            sessionContext,
		cancelSession:         cancelSession,
		histories: []internal_type.MessagePacket{
			internal_type.MessageCreatePacket{
				ContextID:   "msg-user-1",
				MessageRole: "user",
				Text:        "hello",
			},
		},
		metadata: map[string]interface{}{},
		metrics:  map[string]*protos.Metric{},
	}
	requestor.assistant.Id = 101
	requestor.assistantConversation.Id = 303
	requestor.assistantAnalyseExecutors = []internal_type.AnalysisExecutor{
		&recordingAnalysisExecutor{
			order: &order,
			execute: func(internal_type.AnalysisInput) *protos.Metadata {
				return &protos.Metadata{Key: "analysis.summary", Value: "done"}
			},
		},
	}

	packetNames := runFinalizeDataFlow(requestor, "ctx-final-flow")

	assert.Equal(t, []internal_type.PacketName{
		internal_type.PacketNameFinalizeConversationRecordingExecutor,
		internal_type.PacketNameFinalizeSessionRuntime,
		internal_type.PacketNameFinalizeArtifactPushExecutor,
		internal_type.PacketNameExecuteAnalysis,
		internal_type.PacketNameFinalizeConversation,
		internal_type.PacketNameFinalizeAnalysisExecutor,
		internal_type.PacketNameFinalizeAssistant,
		internal_type.PacketNameFinalizationCompleted,
	}, packetNames)
	assert.Equal(t, []string{"analysis-execute", "record-webhook", "analysis-close"}, order)
	require.Len(t, recorder.records, 1)
	webhookRecord, ok := recorder.records[0].(observability.RecordWebhook)
	require.True(t, ok)
	metadataPayload, ok := webhookRecord.Payload["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "done", metadataPayload["analysis.summary"])
	assert.Equal(t, adapter_lifecycle.StateDisconnected, requestor.sessionLifecycle.Current())
}

func TestFinalizeFlow_WithoutAnalysis_RecordsWebhookThenEnds(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	sessionContext, cancelSession := context.WithCancel(context.Background())
	t.Cleanup(cancelSession)
	recorder := &recordingObservabilityRecorder{}
	requestor := &genericRequestor{
		assistant:                 &internal_assistant_entity.Assistant{},
		assistantConversation:     &internal_conversation_entity.AssistantConversation{},
		observabilityRecorder:     recorder,
		messageLifecycle:          adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:          adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateDisconnecting),
		dispatchRoute:             adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:                  requestorChannels,
		sessionCtx:                sessionContext,
		cancelSession:             cancelSession,
		assistantAnalyseExecutors: []internal_type.AnalysisExecutor{},
		metadata:                  map[string]interface{}{},
		metrics:                   map[string]*protos.Metric{},
	}
	requestor.assistant.Id = 101
	requestor.assistantConversation.Id = 303

	packetNames := runFinalizeDataFlow(requestor, "ctx-final-flow-no-analysis")

	assert.Equal(t, []internal_type.PacketName{
		internal_type.PacketNameFinalizeConversationRecordingExecutor,
		internal_type.PacketNameFinalizeSessionRuntime,
		internal_type.PacketNameFinalizeArtifactPushExecutor,
		internal_type.PacketNameExecuteAnalysis,
		internal_type.PacketNameFinalizeConversation,
		internal_type.PacketNameFinalizeAnalysisExecutor,
		internal_type.PacketNameFinalizeAssistant,
		internal_type.PacketNameFinalizationCompleted,
	}, packetNames)
	require.Len(t, recorder.records, 1)
	webhookRecord, ok := recorder.records[0].(observability.RecordWebhook)
	require.True(t, ok)
	assert.Equal(t, observability.ConversationCompleted, webhookRecord.Event)
	assert.Equal(t, adapter_lifecycle.StateDisconnected, requestor.sessionLifecycle.Current())
}

func TestFinalizeFlow_WithoutAnalysisAndWebhookRecorder_EndsWithoutWebhook(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	sessionContext, cancelSession := context.WithCancel(context.Background())
	t.Cleanup(cancelSession)
	requestor := &genericRequestor{
		assistant:                 &internal_assistant_entity.Assistant{},
		assistantConversation:     &internal_conversation_entity.AssistantConversation{},
		messageLifecycle:          adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:          adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateDisconnecting),
		dispatchRoute:             adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:                  requestorChannels,
		sessionCtx:                sessionContext,
		cancelSession:             cancelSession,
		assistantAnalyseExecutors: []internal_type.AnalysisExecutor{},
		metadata:                  map[string]interface{}{},
		metrics:                   map[string]*protos.Metric{},
	}
	requestor.assistant.Id = 101
	requestor.assistantConversation.Id = 303

	packetNames := runFinalizeDataFlow(requestor, "ctx-final-flow-no-webhook")

	assert.Equal(t, []internal_type.PacketName{
		internal_type.PacketNameFinalizeConversationRecordingExecutor,
		internal_type.PacketNameFinalizeSessionRuntime,
		internal_type.PacketNameFinalizeArtifactPushExecutor,
		internal_type.PacketNameExecuteAnalysis,
		internal_type.PacketNameFinalizeConversation,
		internal_type.PacketNameFinalizeAnalysisExecutor,
		internal_type.PacketNameFinalizeAssistant,
		internal_type.PacketNameFinalizationCompleted,
	}, packetNames)
	assert.Nil(t, requestor.observabilityRecorder)
	assert.Equal(t, adapter_lifecycle.StateDisconnected, requestor.sessionLifecycle.Current())
}

func TestHandleError_NonRecoverable_EmitsConversationErrorWebhookRecord(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		streamer:              &streamTestStreamer{},
		assistant:             &internal_assistant_entity.Assistant{},
		assistantConversation: &internal_conversation_entity.AssistantConversation{},
		messageLifecycle:      adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:      adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:         adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:              requestorChannels,
	}
	requestor.assistant.Id = 101
	requestor.assistantConversation.Id = 303

	requestorDispatchHandler{r: requestor}.HandleError(context.Background(), internal_type.InitializationFailedPacket{
		ContextID: "ctx-error-webhook",
		Stage:     internal_type.InitializationStageTextToSpeech,
		Error:     errors.New("tts provider rejected credentials"),
	})

	var webhookPacket internal_type.ObservabilityWebhookRecordPacket
	found := false
	for len(requestor.channels.BackgroundChannel()) > 0 {
		envelope := <-requestor.channels.BackgroundChannel()
		packet, ok := envelope.Pkt.(internal_type.ObservabilityWebhookRecordPacket)
		if !ok {
			continue
		}
		webhookPacket = packet
		found = true
	}

	require.True(t, found, "expected ObservabilityWebhookRecordPacket")
	assert.Equal(t, "ctx-error-webhook", webhookPacket.ContextID)
	assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, webhookPacket.Scope)
	assert.Equal(t, observability.ConversationError, webhookPacket.Record.Event)
	assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(), webhookPacket.Record.Payload["reason"])
	assert.Contains(t, webhookPacket.Record.Payload["message"], "tts provider rejected credentials")
}

func TestHandleInitializeBehavior_RuntimeConfigUnavailable_LogsAndReturns(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		source:           utils.Debugger,
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}

	requestorDispatchHandler{r: requestor}.HandleInitializeBehavior(context.Background(), internal_type.InitializeBehaviorPacket{
		ContextID: "ctx-behavior-missing",
		Config:    &protos.ConversationInitialization{},
	})

	select {
	case envelope := <-requestor.channels.BackgroundChannel():
		_, ok := envelope.Pkt.(internal_type.ObservabilityLogRecordPacket)
		require.True(t, ok, "expected ObservabilityLogRecordPacket, got %T", envelope.Pkt)
	default:
		t.Fatal("expected session runtime initialization failure log")
	}
	assert.Empty(t, requestor.channels.BootstrapChannel())
}
