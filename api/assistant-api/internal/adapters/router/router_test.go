package router

import (
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

type unroutedPacket struct {
	contextID string
}

func (p unroutedPacket) ContextId() string { return p.contextID }
func (p unroutedPacket) PacketName() internal_type.PacketName {
	return internal_type.PacketName("unroutedPacket")
}

func TestClassify(t *testing.T) {
	type testCase struct {
		name      string
		pkt       internal_type.Packet
		wantRoute Route
		wantOK    bool
	}

	cases := []testCase{
		{
			name:      "control",
			pkt:       internal_type.TurnChangePacket{ContextID: "c"},
			wantRoute: RouteControl,
			wantOK:    true,
		},
		{
			name:      "dispatch-policy-control",
			pkt:       internal_type.DispatchPolicyPacket{ContextID: "c"},
			wantRoute: RouteControl,
			wantOK:    true,
		},
		{
			name:      "speech-to-text-start-control",
			pkt:       internal_type.SpeechToTextStartPacket{ContextID: "c"},
			wantRoute: RouteControl,
			wantOK:    true,
		},
		{
			name:      "bootstrap",
			pkt:       internal_type.InitializeAssistantPacket{ContextID: "c"},
			wantRoute: RouteBootstrap,
			wantOK:    true,
		},
		{
			name:      "assistant-executor-bootstrap",
			pkt:       internal_type.InitializeAssistantExecutorPacket{ContextID: "c"},
			wantRoute: RouteBootstrap,
			wantOK:    true,
		},
		{
			name:      "ingress",
			pkt:       internal_type.UserTextReceivedPacket{ContextID: "c", Text: "hi"},
			wantRoute: RouteIngress,
			wantOK:    true,
		},
		{
			name:      "egress",
			pkt:       internal_type.TextToSpeechTextPacket{ContextID: "c", Text: "hi"},
			wantRoute: RouteEgress,
			wantOK:    true,
		},
		{
			name:      "egress-mode-switch-error",
			pkt:       internal_type.ModeSwitchErrorPacket{ContextID: "c"},
			wantRoute: RouteEgress,
			wantOK:    true,
		},
		{
			name:      "background",
			pkt:       internal_type.ObservabilityEventRecordPacket{ContextID: "c"},
			wantRoute: RouteBackground,
			wantOK:    true,
		},
		{
			name:      "background-finalize",
			pkt:       internal_type.FinalizeBehaviorPacket{ContextID: "c"},
			wantRoute: RouteData,
			wantOK:    true,
		},
		{
			name:      "fallback-background-for-unrouted-packet",
			pkt:       unroutedPacket{contextID: "c"},
			wantRoute: RouteBackground,
			wantOK:    false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotRoute := Classify(tc.pkt)
			if gotRoute != tc.wantRoute {
				t.Fatalf("route mismatch: got=%v want=%v", gotRoute, tc.wantRoute)
			}
		})
	}
}

func TestClassifyName_DispatchablePacketNamesAreExplicitlyRouted(t *testing.T) {
	cases := map[internal_type.PacketName]Route{
		internal_type.PacketNameUserTextReceived:                           RouteIngress,
		internal_type.PacketNameUserAudioReceived:                          RouteIngress,
		internal_type.PacketNameSpeechToTextAudio:                          RouteIngress,
		internal_type.PacketNameDenoiseAudio:                               RouteIngress,
		internal_type.PacketNameDenoisedAudio:                              RouteIngress,
		internal_type.PacketNameVadAudio:                                   RouteIngress,
		internal_type.PacketNameVadSpeechActivity:                          RouteIngress,
		internal_type.PacketNameSpeechToText:                               RouteIngress,
		internal_type.PacketNameInterimEndOfSpeech:                         RouteIngress,
		internal_type.PacketNameEndOfSpeech:                                RouteIngress,
		internal_type.PacketNameUserInput:                                  RouteIngress,
		internal_type.PacketNameInterruptionDetected:                       RouteControl,
		internal_type.PacketNameEndOfSpeechInterruption:                    RouteControl,
		internal_type.PacketNameEndOfSpeechAudio:                           RouteIngress,
		internal_type.PacketNameTextToSpeechInterrupt:                      RouteControl,
		internal_type.PacketNameLLMInterrupt:                               RouteControl,
		internal_type.PacketNameDispatchPolicy:                             RouteControl,
		internal_type.PacketNameSpeechToTextEnd:                            RouteControl,
		internal_type.PacketNameSpeechToTextStart:                          RouteControl,
		internal_type.PacketNameTurnChange:                                 RouteControl,
		internal_type.PacketNameLLMResponseDelta:                           RouteEgress,
		internal_type.PacketNameLLMResponseDone:                            RouteEgress,
		internal_type.PacketNameSpeechToTextError:                          RouteEgress,
		internal_type.PacketNameLLMError:                                   RouteEgress,
		internal_type.PacketNameTextToSpeechError:                          RouteEgress,
		internal_type.PacketNameModeSwitchError:                            RouteEgress,
		internal_type.PacketNameInjectMessage:                              RouteEgress,
		internal_type.PacketNameStartIdleTimeout:                           RouteEgress,
		internal_type.PacketNameStopIdleTimeout:                            RouteEgress,
		internal_type.PacketNameIdleTimeoutExpired:                         RouteEgress,
		internal_type.PacketNameUnclearInputExpired:                        RouteEgress,
		internal_type.PacketNameMaxSessionExpired:                          RouteEgress,
		internal_type.PacketNameTextToSpeechText:                           RouteEgress,
		internal_type.PacketNameTextToSpeechDone:                           RouteEgress,
		internal_type.PacketNameTextToSpeechAudio:                          RouteEgress,
		internal_type.PacketNameTextToSpeechEnd:                            RouteEgress,
		internal_type.PacketNameLLMToolCall:                                RouteEgress,
		internal_type.PacketNameLLMToolResult:                              RouteIngress,
		internal_type.PacketNameRecordUserAudio:                            RouteData,
		internal_type.PacketNameRecordAssistantAudio:                       RouteData,
		internal_type.PacketNameConversationRecordingCompleted:             RouteData,
		internal_type.PacketNameMessageCreate:                              RouteData,
		internal_type.PacketNameToolLogCreate:                              RouteData,
		internal_type.PacketNameToolLogUpdate:                              RouteData,
		internal_type.PacketNameHTTPLogCreate:                              RouteData,
		internal_type.PacketNameInitializeAssistant:                        RouteBootstrap,
		internal_type.PacketNameInitializeConversation:                     RouteBootstrap,
		internal_type.PacketNameInitializeSessionRuntime:                   RouteBootstrap,
		internal_type.PacketNameInitializeConversationRecordingExecutor:    RouteBootstrap,
		internal_type.PacketNameInitializeArtifactPushExecutor:             RouteBootstrap,
		internal_type.PacketNameInitializeAnalysisExecutor:                 RouteBootstrap,
		internal_type.PacketNameInitializeAuthentication:                   RouteBootstrap,
		internal_type.PacketNameExecuteAuthentication:                      RouteBootstrap,
		internal_type.PacketNameSessionAuthenticationSucceeded:             RouteBootstrap,
		internal_type.PacketNameSessionAuthenticationFailed:                RouteBootstrap,
		internal_type.PacketNameInitializeSpeechToText:                     RouteBootstrap,
		internal_type.PacketNameInitializeAssistantExecutor:                RouteBootstrap,
		internal_type.PacketNameInitializeTextToSpeech:                     RouteBootstrap,
		internal_type.PacketNameInitializeVoiceActivityDetection:           RouteBootstrap,
		internal_type.PacketNameInitializeEndOfSpeech:                      RouteBootstrap,
		internal_type.PacketNameInitializeDenoise:                          RouteBootstrap,
		internal_type.PacketNameInitializeBehavior:                         RouteBootstrap,
		internal_type.PacketNameInitializationCompleted:                    RouteBootstrap,
		internal_type.PacketNameInitializationFailed:                       RouteBootstrap,
		internal_type.PacketNameInitializeInboundDispatcher:                RouteBootstrap,
		internal_type.PacketNameFinalizeInboundDispatcher:                  RouteBootstrap,
		internal_type.PacketNameModeSwitchRequested:                        RouteBootstrap,
		internal_type.PacketNameModeSwitchCompleted:                        RouteBootstrap,
		internal_type.PacketNameModeSwitchInitializeSpeechToText:           RouteBootstrap,
		internal_type.PacketNameModeSwitchInitializeTextToSpeech:           RouteBootstrap,
		internal_type.PacketNameModeSwitchInitializeVoiceActivityDetection: RouteBootstrap,
		internal_type.PacketNameModeSwitchInitializeEndOfSpeech:            RouteBootstrap,
		internal_type.PacketNameModeSwitchInitializeDenoise:                RouteBootstrap,
		internal_type.PacketNameModeSwitchFinalizeEndOfSpeech:              RouteBootstrap,
		internal_type.PacketNameModeSwitchFinalizeDenoise:                  RouteBootstrap,
		internal_type.PacketNameModeSwitchFinalizeVoiceActivityDetection:   RouteBootstrap,
		internal_type.PacketNameModeSwitchFinalizeTextToSpeech:             RouteBootstrap,
		internal_type.PacketNameModeSwitchFinalizeSpeechToText:             RouteBootstrap,
		internal_type.PacketNameFinalizeBehavior:                           RouteData,
		internal_type.PacketNameFinalizeEndOfSpeech:                        RouteData,
		internal_type.PacketNameFinalizeVoiceActivityDetection:             RouteData,
		internal_type.PacketNameFinalizeTextToSpeech:                       RouteData,
		internal_type.PacketNameFinalizeSpeechToText:                       RouteData,
		internal_type.PacketNameFinalizeAuthentication:                     RouteData,
		internal_type.PacketNameFinalizeConversationRecordingExecutor:      RouteData,
		internal_type.PacketNameFinalizeSessionRuntime:                     RouteData,
		internal_type.PacketNameFinalizeArtifactPushExecutor:               RouteData,
		internal_type.PacketNameExecuteAnalysis:                            RouteData,
		internal_type.PacketNameFinalizeConversation:                       RouteData,
		internal_type.PacketNameFinalizeAnalysisExecutor:                   RouteData,
		internal_type.PacketNameFinalizeAssistant:                          RouteData,
		internal_type.PacketNameFinalizationCompleted:                      RouteData,
		internal_type.PacketNameObservabilityLogRecord:                     RouteBackground,
		internal_type.PacketNameObservabilityEventRecord:                   RouteBackground,
		internal_type.PacketNameObservabilityMetricRecord:                  RouteBackground,
		internal_type.PacketNameObservabilityMetadataRecord:                RouteBackground,
		internal_type.PacketNameObservabilityUsageRecord:                   RouteBackground,
		internal_type.PacketNameObservabilityWebhookRecord:                 RouteBackground,
	}

	for packetName, wantRoute := range cases {
		packetName := packetName
		wantRoute := wantRoute
		t.Run(string(packetName), func(t *testing.T) {
			if gotRoute := ClassifyName(packetName); gotRoute != wantRoute {
				t.Fatalf("route mismatch: got=%v want=%v", gotRoute, wantRoute)
			}
		})
	}
}
