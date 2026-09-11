package node

import (
	"context"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/state"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

type packetRecordingCommunication struct {
	internal_type.Communication
	packets []internal_type.Packet
}

func (communication *packetRecordingCommunication) OnPacket(_ context.Context, packets ...internal_type.Packet) error {
	communication.packets = append(communication.packets, packets...)
	return nil
}

func TestMessageHandlerEmitsDeltaAndReturnsResponseText(t *testing.T) {
	communication := &packetRecordingCommunication{}
	result, err := MessageHandler{}.Execute(context.Background(), Request{
		ContextID: "ctx-1",
		Node: schema.Node{
			ID:     "message-1",
			Type:   schema.NodeTypeMessage,
			Config: utils.Option{"message": "Hello"},
		},
		RuntimeState:  state.NewRuntimeState("message-1"),
		Communication: communication,
	})
	require.NoError(t, err)

	require.Equal(t, []string{"response", "next"}, result.RouteHandles)
	require.Equal(t, "Hello", result.ResponseText)
	require.Len(t, communication.packets, 1)
	delta, ok := communication.packets[0].(internal_type.LLMResponseDeltaPacket)
	require.True(t, ok)
	require.Equal(t, "Hello", delta.Text)
}

func TestTransferHandlerEmitsDeltaAndReturnsResponseText(t *testing.T) {
	communication := &packetRecordingCommunication{}
	result, err := TransferHandler{}.Execute(context.Background(), Request{
		ContextID: "ctx-1",
		Node: schema.Node{
			ID:   "transfer-1",
			Type: schema.NodeTypeTransfer,
			Config: utils.Option{
				"transfer_message": "Connecting you now",
				"transfer_to":      "+15551234567",
			},
		},
		RuntimeState:  state.NewRuntimeState("transfer-1"),
		Communication: communication,
	})
	require.NoError(t, err)

	require.True(t, result.Terminal)
	require.Equal(t, "Connecting you now", result.ResponseText)
	require.Len(t, communication.packets, 2)
	delta, ok := communication.packets[0].(internal_type.LLMResponseDeltaPacket)
	require.True(t, ok)
	require.Equal(t, "Connecting you now", delta.Text)
	toolCall, ok := communication.packets[1].(internal_type.LLMToolCallPacket)
	require.True(t, ok)
	require.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, toolCall.Action)
}
