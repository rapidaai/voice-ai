package internal_exotel_telephony

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
	"github.com/rapidaai/protos"
)

type exotelWebsocketStreamer struct {
	logger     commons.Logger
	conn       *websocket.Conn
	ctx        context.Context
	cancelFunc context.CancelFunc

	assistant               *protos.AssistantDefinition
	version                 string
	assistantConversationId uint64
	streamSid               string

	//
	// mutex
	audioBufferLock   sync.Mutex
	inputAudioBuffer  *bytes.Buffer
	outputAudioBuffer *bytes.Buffer
	encoder           *base64.Encoding
}
type ExotelMediaEvent struct {
	Event     string       `json:"event"`
	StreamSid string       `json:"stream_sid"`
	Media     *ExotelMedia `json:"media,omitempty"`
}

type ExotelMedia struct {
	Payload string `json:"payload"`
}

func NewExotelWebsocketStreamer(
	logger commons.Logger,
	connection *websocket.Conn,
	assistantId uint64,
	version string,
	conversationId uint64,
) internal_streamers.Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	return &exotelWebsocketStreamer{
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

		//
		inputAudioBuffer:  new(bytes.Buffer),
		outputAudioBuffer: new(bytes.Buffer),
		encoder:           base64.StdEncoding,
	}
}

func (exotel *exotelWebsocketStreamer) Context() context.Context {
	return exotel.ctx
}

func (exotel *exotelWebsocketStreamer) Recv() (*protos.AssistantMessagingRequest, error) {
	if exotel.conn == nil {
		exotel.logger.Error("WebSocket connection is nil")
		return nil, io.EOF
	}

	_, message, err := exotel.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			exotel.logger.Error("Unexpected websocket close error", "error", err.Error())
		} else {
			exotel.logger.Error("Failed to read message from WebSocket", "error", err.Error())
		}
		exotel.cancelFunc()
		exotel.conn = nil
		return nil, io.EOF
	}

	var mediaEvent ExotelMediaEvent
	if err := json.Unmarshal(message, &mediaEvent); err != nil {
		exotel.logger.Error("Failed to unmarshal Exotel media event", "error", err.Error())
		return nil, nil
	}

	if exotel.streamSid == "" && mediaEvent.StreamSid != "" {
		exotel.streamSid = mediaEvent.StreamSid
		exotel.logger.Debug("Captured Exotel streamSid", "streamSid", exotel.streamSid)
	}
	switch mediaEvent.Event {
	case "start":
		return nil, nil

	case "media":
		return exotel.handleMediaEvent(mediaEvent)

	case "dtmf":
		// Handle DTMF if needed
		return nil, nil

	case "stop":
		// exotel.logger.Info("Exotel stream stopped", "reason", mediaEvent.Stop.Reason)
		exotel.cancelFunc()
		return nil, io.EOF

	default:
		exotel.logger.Warn("Unhandled Exotel event", "event", mediaEvent.Event)
		return nil, nil
	}
}
func (exotel *exotelWebsocketStreamer) handleMediaEvent(mediaEvent ExotelMediaEvent) (*protos.AssistantMessagingRequest, error) {
	payloadBytes, err := exotel.encoder.DecodeString(mediaEvent.Media.Payload)
	if err != nil {
		exotel.logger.Warn("Failed to decode media payload", "error", err.Error())
		return nil, nil
	}

	exotel.audioBufferLock.Lock()
	defer exotel.audioBufferLock.Unlock()

	// 1ms 8 bytes @ 8kHz µ-law mono 60ms of audio as silero can't process smaller chunk for mulaw
	exotel.inputAudioBuffer.Write(payloadBytes)
	const bufferSizeThreshold = 32 * 60

	if exotel.inputAudioBuffer.Len() >= bufferSizeThreshold {
		audioRequest := exotel.BuildVoiceRequest(exotel.inputAudioBuffer.Bytes())
		exotel.inputAudioBuffer.Reset()
		return audioRequest, nil
	}

	return nil, nil
}

func (exotel *exotelWebsocketStreamer) BuildVoiceRequest(audioData []byte) *protos.AssistantMessagingRequest {
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

func (exotel *exotelWebsocketStreamer) Send(response *protos.AssistantMessagingResponse) error {
	switch data := response.GetData().(type) {
	case *protos.AssistantMessagingResponse_Assistant:
		switch content := data.Assistant.Message.(type) {
		case *protos.AssistantConversationAssistantMessage_Audio:
			//1ms 32  10ms 320byte @ 16000Hz, 16-bit mono PCM = 640 bytes
			// Each message needs to be a 20ms sample of audio.
			// At 8kHz the message should be 320 bytes.
			// At 16kHz the message should be 640 bytes.
			bufferSizeThreshold := 32 * 20
			audioData := content.Audio.GetContent()

			// Use vng.audioBuffer to handle pending data across calls
			exotel.audioBufferLock.Lock()
			defer exotel.audioBufferLock.Unlock()

			// Append incoming audio data to the buffer
			exotel.outputAudioBuffer.Write(audioData)
			// Process and send chunks of 640 bytes
			for exotel.outputAudioBuffer.Len() >= bufferSizeThreshold && exotel.streamSid != "" {
				chunk := exotel.outputAudioBuffer.Next(bufferSizeThreshold) // Get and remove the next 640 bytes
				if err := exotel.sendingExotelMessage("media", map[string]interface{}{
					"payload": exotel.encoder.EncodeToString(chunk),
				}); err != nil {
					exotel.logger.Error("Failed to send audio chunk", "error", err.Error())
					return err
				}
			}

			// If response is marked as completed, flush any remaining audio in the buffer
			if data.Assistant.GetCompleted() && exotel.outputAudioBuffer.Len() > 0 {
				remainingChunk := exotel.outputAudioBuffer.Bytes()
				if err := exotel.sendingExotelMessage("media", map[string]interface{}{
					"payload": exotel.encoder.EncodeToString(remainingChunk),
				}); err != nil {
					exotel.logger.Errorf("Failed to send final audio chunk", "error", err.Error())
					return err
				}
				exotel.outputAudioBuffer.Reset() // Clear the buffer after flushing
			}
		}
	case *protos.AssistantMessagingResponse_Interruption:
		exotel.audioBufferLock.Lock()
		defer exotel.audioBufferLock.Unlock()
		exotel.outputAudioBuffer.Reset() // Clear the buffer after flushing
		if err := exotel.sendingExotelMessage("clear", nil); err != nil {
			exotel.logger.Errorf("Error sending clear command:", err)
		}
	case *protos.AssistantMessagingResponse_Action:
		if data.Action.GetAction() == protos.AssistantConversationAction_END_CONVERSATION {
			if err := exotel.conn.Close(); err != nil {
				// terminate the conversation as end tool call is triggered
				exotel.logger.Errorf("Error disconnecting command:", err)
			}
		}
	}
	return nil
}

func (exotel *exotelWebsocketStreamer) sendingExotelMessage(
	eventType string,
	mediaData map[string]interface{}) error {
	if exotel.conn == nil || exotel.streamSid == "" {
		return nil
	}
	message := map[string]interface{}{
		"event":     eventType,
		"streamSid": exotel.streamSid,
	}
	if mediaData != nil {
		message["media"] = mediaData
	}

	exotelMessageJSON, err := json.Marshal(message)
	if err != nil {
		return exotel.handleError("Failed to marshal Exotel message", err)
	}

	if err := exotel.conn.WriteMessage(websocket.TextMessage, exotelMessageJSON); err != nil {
		return exotel.handleError("Failed to send message to Exotel", err)
	}

	return nil
}

func (exo *exotelWebsocketStreamer) handleError(message string, err error) error {
	exo.logger.Error(message, "error", err.Error())
	return err
}
