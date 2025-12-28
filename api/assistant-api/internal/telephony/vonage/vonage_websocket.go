package internal_vonage_telephony

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"sync"

	"github.com/gorilla/websocket"
	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
	"github.com/rapidaai/pkg/commons"
	protos "github.com/rapidaai/protos"
)

type vonageWebsocketStreamer struct {
	logger     commons.Logger
	conn       *websocket.Conn
	ctx        context.Context
	cancelFunc context.CancelFunc

	assistant               *protos.AssistantDefinition
	version                 string
	assistantConversationId uint64

	inputAudioBuffer  *bytes.Buffer
	outputAudioBuffer *bytes.Buffer

	// mutex
	audioBufferLock sync.Mutex

	//
	encoder *base64.Encoding
}

type VonageMediaEvent struct {
	Event string `json:"event"`
	Media struct {
		Track     string `json:"track"`
		Chunk     string `json:"chunk"`
		Timestamp string `json:"timestamp"`
		Payload   string `json:"payload"`
	} `json:"media"`
	StreamSid string `json:"streamSid"`
}

func NewVonageWebsocketStreamer(
	logger commons.Logger,
	connection *websocket.Conn,
	assistantId uint64,
	version string,
	conversationId uint64,
) internal_streamers.Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	return &vonageWebsocketStreamer{
		logger:     logger,
		conn:       connection,
		ctx:        ctx,
		cancelFunc: cancel,
		assistant: &protos.AssistantDefinition{
			AssistantId: assistantId,
			Version:     version,
		},
		version:                 version,
		assistantConversationId: conversationId,
		inputAudioBuffer:        new(bytes.Buffer),
		outputAudioBuffer:       new(bytes.Buffer),
		encoder:                 base64.StdEncoding,
	}
}

func (vng *vonageWebsocketStreamer) Context() context.Context {
	return vng.ctx
}

func (vng *vonageWebsocketStreamer) Recv() (*protos.AssistantMessagingRequest, error) {
	if vng.conn == nil {
		return nil, vng.handleError("WebSocket connection is nil", io.EOF)
	}
	messageType, message, err := vng.conn.ReadMessage()
	if err != nil {
		return nil, vng.handleWebSocketError(err)
	}

	switch messageType {
	case websocket.TextMessage:

		var textEvent map[string]interface{}
		if err := json.Unmarshal(message, &textEvent); err != nil {
			vng.logger.Error("Failed to unmarshal text event", "error", err.Error())
			return nil, err
		}

		// Example for handling a specific text message event, modify as needed
		if textEvent["event"] == "stop" {
			vng.logger.Info("WebSocket stream stopped")
			vng.cancelFunc()
			return nil, io.EOF
		}

	case websocket.BinaryMessage:
		vng.audioBufferLock.Lock()
		defer vng.audioBufferLock.Unlock()

		vng.inputAudioBuffer.Write(message)
		const bufferSizeThreshold = 32 * 60

		if vng.inputAudioBuffer.Len() >= bufferSizeThreshold {
			audioRequest := vng.BuildVoiceRequest(vng.inputAudioBuffer.Bytes())
			vng.inputAudioBuffer.Reset()
			return audioRequest, nil
		}

		// send back the audio

	default:
		vng.logger.Warn("Unhandled message type", "type", messageType)
		return nil, nil
	}

	// No actionable request generated
	return nil, nil
}

func (vng *vonageWebsocketStreamer) BuildVoiceRequest(audioData []byte) *protos.AssistantMessagingRequest {
	return &protos.AssistantMessagingRequest{
		Request: &protos.AssistantMessagingRequest_Message{
			Message: &protos.AssistantConversationUserMessage{
				Message: &protos.AssistantConversationUserMessage_Audio{
					Audio: &protos.AssistantConversationMessageAudioContent{
						Content: audioData,
					},
				},
			},
		},
	}
}

func (vng *vonageWebsocketStreamer) Send(response *protos.AssistantMessagingResponse) error {
	if vng.conn == nil {
		return nil
	}
	switch data := response.GetData().(type) {
	case *protos.AssistantMessagingResponse_Assistant:
		switch content := data.Assistant.Message.(type) {
		case *protos.AssistantConversationAssistantMessage_Audio:
			//	1ms 32  10ms 320byte @ 16000Hz, 16-bit mono PCM = 640 bytes
			// Each message needs to be a 20ms sample of audio.
			// At 8kHz the message should be 320 bytes.
			// At 16kHz the message should be 640 bytes.
			bufferSizeThreshold := 32 * 20
			audioData := content.Audio.GetContent()

			// Use vng.audioBuffer to handle pending data across calls
			vng.audioBufferLock.Lock()
			defer vng.audioBufferLock.Unlock()

			// Append incoming audio data to the buffer
			vng.outputAudioBuffer.Write(audioData)
			// Process and send chunks of 640 bytes
			for vng.outputAudioBuffer.Len() >= bufferSizeThreshold {
				chunk := vng.outputAudioBuffer.Next(bufferSizeThreshold) // Get and remove the next 640 bytes
				if err := vng.conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
					vng.logger.Error("Failed to send audio chunk", "error", err.Error())
					return err
				}
			}

			// If response is marked as completed, flush any remaining audio in the buffer
			if data.Assistant.GetCompleted() && vng.outputAudioBuffer.Len() > 0 {
				remainingChunk := vng.outputAudioBuffer.Bytes()
				if err := vng.conn.WriteMessage(websocket.BinaryMessage, remainingChunk); err != nil {
					vng.logger.Errorf("Failed to send final audio chunk", "error", err.Error())
					return err
				}
				vng.outputAudioBuffer.Reset() // Clear the buffer after flushing
			}
		}
	case *protos.AssistantMessagingResponse_Interruption:
		vng.logger.Debugf("clearing action")
		vng.audioBufferLock.Lock()
		defer vng.audioBufferLock.Unlock()
		vng.outputAudioBuffer.Reset()

		// Clear the buffer after flushing
		err := vng.conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"clear"}`))
		if err != nil {
			vng.logger.Errorf("Error sending clear command:", err)
		}
	case *protos.AssistantMessagingResponse_Action:
		if data.Action.GetAction() == protos.AssistantConversationAction_END_CONVERSATION {
			if err := vng.conn.Close(); err != nil {
				vng.logger.Errorf("Error disconnecting command:", err)
			}
		}
	}
	return nil
}
func (vng *vonageWebsocketStreamer) handleError(message string, err error) error {
	vng.logger.Error(message, "error", err.Error())
	return err
}

func (vng *vonageWebsocketStreamer) handleWebSocketError(err error) error {
	if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
		vng.logger.Error("Unexpected websocket close error", "error", err.Error())
	} else {
		vng.logger.Error("Failed to read message from WebSocket", "error", err.Error())
	}
	vng.cancelFunc()
	vng.conn = nil
	return io.EOF
}
