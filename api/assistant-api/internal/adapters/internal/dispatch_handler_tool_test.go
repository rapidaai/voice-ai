package adapter_internal

import (
	"context"
	"errors"
	"testing"
	"time"

	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type toolDispatchTestExecutor struct {
	packets chan internal_type.Packet
	err     error
}

func (*toolDispatchTestExecutor) Name() string {
	return "tool-dispatch-test"
}

func (executor *toolDispatchTestExecutor) Execute(_ context.Context, _ internal_type.Communication, packet internal_type.Packet) error {
	executor.packets <- packet
	return executor.err
}

func (*toolDispatchTestExecutor) Close(context.Context) error {
	return nil
}

func TestHandleLLMToolCall_DoesNotInterruptSpeech(test *testing.T) {
	for _, testCase := range []struct {
		name      string
		arguments map[string]string
	}{
		{name: "progress message", arguments: map[string]string{"message": "Let me send an OTP now."}},
		{name: "no message"},
		{name: "empty message", arguments: map[string]string{"message": ""}},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			requestor := newInterruptionTestRequestor("")
			executor := &toolDispatchTestExecutor{packets: make(chan internal_type.Packet, 2)}
			requestor.assistantExecutor = executor
			handler := requestorDispatchHandler{r: requestor}
			packet := internal_type.LLMToolCallPacket{
				ContextID: "ctx-active",
				ToolID:    "otp-tool",
				Name:      "send_otp",
				Arguments: testCase.arguments,
			}

			handler.HandleLLMToolCall(context.Background(), packet)
			handler.HandlePlaybackCompleted(context.Background(), internal_type.PlaybackCompletedPacket{ContextID: packet.ContextID})

			assert.Empty(test, drainControlPackets(requestor))
			assert.Equal(test, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
			if message := testCase.arguments["message"]; message != "" {
				assert.Equal(test, []internal_type.Packet{
					internal_type.TextToSpeechTextPacket{ContextID: packet.ContextID, Text: message},
				}, drainEgressPackets(requestor))
			} else {
				assert.Empty(test, drainEgressPackets(requestor))
			}

			streamer := requestor.streamer.(*streamTestStreamer)
			require.Len(test, streamer.sent, 1)
			call, ok := streamer.sent[0].(*protos.ConversationToolCall)
			require.True(test, ok)
			assert.Equal(test, packet.ContextID, call.Id)
			assert.Equal(test, packet.ToolID, call.ToolId)
			assert.Equal(test, packet.Name, call.Name)
			assert.Equal(test, packet.Arguments, call.Args)
			expected := []internal_type.Packet{packet}
			if message := testCase.arguments["message"]; message != "" {
				expected = append(expected, internal_type.InjectMessagePacket{ContextID: packet.ContextID, Text: message, Interim: true})
			}
			var executed []internal_type.Packet
			for range expected {
				select {
				case received := <-executor.packets:
					executed = append(executed, received)
				case <-time.After(time.Second):
					test.Fatal("tool call was not forwarded to the assistant executor")
				}
			}
			assert.ElementsMatch(test, expected, executed)
		})
	}
}

func TestToolSpeechPrecedesFinalProducerBoundary(t *testing.T) {
	requestor := newInterruptionTestRequestor("")
	handler := requestorDispatchHandler{r: requestor}
	handler.HandleLLMToolCall(context.Background(), internal_type.LLMToolCallPacket{ContextID: "ctx-active", Arguments: map[string]string{"message": "Checking now"}})
	handler.HandleLLMResponseDone(context.Background(), internal_type.LLMResponseDonePacket{ContextID: "ctx-active", Text: "Done"})
	assert.Equal(t, []internal_type.Packet{
		internal_type.TextToSpeechTextPacket{ContextID: "ctx-active", Text: "Checking now"},
		internal_type.TextToSpeechDonePacket{ContextID: "ctx-active", Text: "Done"},
	}, drainEgressPackets(requestor))
}

func TestHandleLLMToolResult_DoesNotInterruptSpeech(test *testing.T) {
	for _, testCase := range []struct {
		name   string
		result map[string]string
		err    error
	}{
		{name: "success", result: map[string]string{"status": "sent"}},
		{name: "tool failure", result: map[string]string{"error": "OTP delivery failed"}},
		{name: "empty result"},
		{name: "executor failure", err: errors.New("tool result processing failed")},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			requestor := newInterruptionTestRequestor("")
			executor := &toolDispatchTestExecutor{
				packets: make(chan internal_type.Packet, 1),
				err:     testCase.err,
			}
			requestor.assistantExecutor = executor
			handler := requestorDispatchHandler{r: requestor}
			packet := internal_type.LLMToolResultPacket{
				ContextID: "ctx-active",
				ToolID:    "otp-tool",
				Name:      "send_otp",
				Result:    testCase.result,
			}

			handler.HandleLLMToolResult(context.Background(), packet)
			handler.HandlePlaybackCompleted(context.Background(), internal_type.PlaybackCompletedPacket{ContextID: packet.ContextID})

			assert.Empty(test, drainControlPackets(requestor))
			assert.Equal(test, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
			for _, packet := range drainEgressPackets(requestor) {
				if typed, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
					test.Fatalf("tool result should not emit idle timeout packet: %+v", typed)
				}
			}
			assert.Empty(test, requestor.streamer.(*streamTestStreamer).sent)
			select {
			case executed := <-executor.packets:
				assert.Equal(test, packet, executed)
			case <-time.After(time.Second):
				test.Fatal("tool result was not forwarded to the assistant executor")
			}
			if testCase.err != nil {
				require.Eventually(test, func() bool {
					for _, emitted := range drainBackgroundPackets(requestor) {
						log, ok := emitted.(internal_type.ObservabilityLogRecordPacket)
						if ok && log.Record.Level == observability.LevelError {
							assert.Equal(test, packet.ContextID, log.ContextID)
							assert.Equal(test, packet.ToolID, log.Record.Attributes["tool_id"])
							assert.Equal(test, testCase.err.Error(), log.Record.Attributes["error"])
							return true
						}
					}
					return false
				}, time.Second, time.Millisecond)
			}
		})
	}
}
