package internal_twilio_telephony

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
	"github.com/rapidaai/pkg/commons"
	protos "github.com/rapidaai/protos"
)

type twilioWebsocketStreamer struct {
	logger     commons.Logger
	conn       *websocket.Conn
	ctx        context.Context
	cancelFunc context.CancelFunc

	assistant               *protos.AssistantDefinition
	version                 string
	assistantConversationId uint64
	streamSid               string

	inputAudioBuffer  *bytes.Buffer
	outputAudioBuffer *bytes.Buffer

	// mutex
	audioBufferLock sync.Mutex

	//
	encoder *base64.Encoding
}

type TwilioMediaEvent struct {
	Event string `json:"event"`
	Media struct {
		Track     string `json:"track"`
		Chunk     string `json:"chunk"`
		Timestamp string `json:"timestamp"`
		Payload   string `json:"payload"`
	} `json:"media"`
	StreamSid string `json:"streamSid"`
}

func NewTwilioWebsocketStreamer(
	logger commons.Logger,
	connection *websocket.Conn,
	assistantId uint64,
	version string,
	conversationId uint64,
) internal_streamers.Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	return &twilioWebsocketStreamer{
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

func (tws *twilioWebsocketStreamer) Context() context.Context {
	return tws.ctx
}

func (tws *twilioWebsocketStreamer) Recv() (*protos.AssistantMessagingRequest, error) {
	if tws.conn == nil {
		return nil, tws.handleError("WebSocket connection is nil", io.EOF)
	}
	_, message, err := tws.conn.ReadMessage()
	if err != nil {
		return nil, tws.handleWebSocketError(err)
	}

	var mediaEvent TwilioMediaEvent
	if err := json.Unmarshal(message, &mediaEvent); err != nil {
		tws.logger.Error("Failed to unmarshal Twilio media event", "error", err.Error())
		return nil, nil
	}
	switch mediaEvent.Event {
	case "connected":
		return tws.handleConnectEvent()
	case "start":
		tws.handleStartEvent(mediaEvent)
		return nil, nil
	case "media":
		return tws.handleMediaEvent(mediaEvent)
	case "stop":
		tws.logger.Info("Twilio stream stopped")
		tws.cancelFunc()
		return nil, io.EOF
	default:
		tws.logger.Warn("Unhandled Twilio event", "event", mediaEvent.Event)
		return nil, nil
	}
}

func (tws *twilioWebsocketStreamer) captureStreamSid(streamSid string) {
	if tws.streamSid == "" && streamSid != "" {
		tws.streamSid = streamSid
		tws.logger.Debug("Captured Twilio streamSid", "streamSid", tws.streamSid)
	}
}

// start event contains streamSid to be used for subsequent media messages
func (tws *twilioWebsocketStreamer) handleStartEvent(mediaEvent TwilioMediaEvent) {
	tws.streamSid = mediaEvent.StreamSid
}

// when exotel is connected then connect the assistant
func (tws *twilioWebsocketStreamer) handleConnectEvent() (*protos.AssistantMessagingRequest, error) {
	return &protos.AssistantMessagingRequest{
		Request: &protos.AssistantMessagingRequest_Configuration{
			Configuration: &protos.AssistantConversationConfiguration{
				AssistantConversationId: tws.assistantConversationId,
				Assistant: &protos.AssistantDefinition{
					AssistantId: tws.assistant.AssistantId,
					Version:     "latest",
				},
				InputConfig: &protos.StreamConfig{
					Audio: internal_audio.NewMulaw8khzMonoAudioConfig(),
				},
				OutputConfig: &protos.StreamConfig{
					Audio: internal_audio.NewMulaw8khzMonoAudioConfig(),
				},
			},
		}}, nil
}

func (tws *twilioWebsocketStreamer) handleMediaEvent(mediaEvent TwilioMediaEvent) (*protos.AssistantMessagingRequest, error) {
	payloadBytes, err := tws.encoder.DecodeString(mediaEvent.Media.Payload)
	if err != nil {
		tws.logger.Warn("Failed to decode media payload", "error", err.Error())
		return nil, nil
	}

	tws.audioBufferLock.Lock()
	defer tws.audioBufferLock.Unlock()

	// 1ms 8 bytes @ 8kHz µ-law mono 60ms of audio as silero can't process smaller chunk for mulaw
	tws.inputAudioBuffer.Write(payloadBytes)
	const bufferSizeThreshold = 8 * 60

	if tws.inputAudioBuffer.Len() >= bufferSizeThreshold {
		audioRequest := tws.buildVoiceRequest(tws.inputAudioBuffer.Bytes())
		tws.inputAudioBuffer.Reset()
		return audioRequest, nil
	}

	return nil, nil
}

func (tws *twilioWebsocketStreamer) buildVoiceRequest(audioData []byte) *protos.AssistantMessagingRequest {
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

func (tws *twilioWebsocketStreamer) Send(response *protos.AssistantMessagingResponse) error {

	switch data := response.GetData().(type) {
	case *protos.AssistantMessagingResponse_Assistant:
		switch content := data.Assistant.Message.(type) {
		case *protos.AssistantConversationAssistantMessage_Audio:
			//1ms 32  10ms 320byte @ 16000Hz, 16-bit mono PCM = 640 bytes
			// Each message needs to be a 20ms sample of audio.
			// At 8kHz the message should be 320 bytes.
			// At 16kHz the message should be 640 bytes.
			bufferSizeThreshold := 8 * 20
			audioData := content.Audio.GetContent()

			// Use vng.audioBuffer to handle pending data across calls
			tws.audioBufferLock.Lock()
			defer tws.audioBufferLock.Unlock()

			// Append incoming audio data to the buffer
			tws.outputAudioBuffer.Write(audioData)
			// Process and send chunks of 640 bytes
			for tws.outputAudioBuffer.Len() >= bufferSizeThreshold && tws.streamSid != "" {
				chunk := tws.outputAudioBuffer.Next(bufferSizeThreshold) // Get and remove the next 640 bytes
				if err := tws.sendTwilioMessage("media", map[string]interface{}{
					"payload": tws.encoder.EncodeToString(chunk),
				}); err != nil {
					tws.logger.Error("Failed to send audio chunk", "error", err.Error())
					return err
				}
			}

			// If response is marked as completed, flush any remaining audio in the buffer
			if data.Assistant.GetCompleted() && tws.outputAudioBuffer.Len() > 0 {
				remainingChunk := tws.outputAudioBuffer.Bytes()
				if err := tws.sendTwilioMessage("media", map[string]interface{}{
					"payload": tws.encoder.EncodeToString(remainingChunk),
				}); err != nil {
					tws.logger.Errorf("Failed to send final audio chunk", "error", err.Error())
					return err
				}
				tws.outputAudioBuffer.Reset() // Clear the buffer after flushing
			}
		}
	case *protos.AssistantMessagingResponse_Interruption:
		tws.logger.Debugf("clearing action")
		tws.audioBufferLock.Lock()
		defer tws.audioBufferLock.Unlock()
		tws.outputAudioBuffer.Reset() // Clear the buffer after flushing
		err := tws.sendTwilioMessage("clear", nil)
		if err != nil {
			tws.logger.Errorf("Error sending clear command:", err)
		}
	case *protos.AssistantMessagingResponse_Action:
		if data.Action.GetAction() == protos.AssistantConversationAction_END_CONVERSATION {
			if err := tws.conn.Close(); err != nil {
				tws.logger.Errorf("Error disconnecting command:", err)
			}
		}
	}
	return nil
}

func (tws *twilioWebsocketStreamer) sendTwilioMessage(
	eventType string,
	mediaData map[string]interface{}) error {
	if tws.conn == nil || tws.streamSid == "" {
		return nil
	}
	message := map[string]interface{}{
		"event":     eventType,
		"streamSid": tws.streamSid,
	}
	if mediaData != nil {
		message["media"] = mediaData
	}

	twilioMessageJSON, err := json.Marshal(message)
	if err != nil {
		return tws.handleError("Failed to marshal Twilio message", err)
	}

	if err := tws.conn.WriteMessage(websocket.TextMessage, twilioMessageJSON); err != nil {
		return tws.handleError("Failed to send message to Twilio", err)
	}

	return nil
}

func (tws *twilioWebsocketStreamer) handleError(message string, err error) error {
	tws.logger.Error(message, "error", err.Error())
	return err
}

func (tws *twilioWebsocketStreamer) handleWebSocketError(err error) error {
	if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
		tws.logger.Error("Unexpected websocket close error", "error", err.Error())
	} else {
		tws.logger.Error("Failed to read message from WebSocket", "error", err.Error())
	}
	tws.cancelFunc()
	tws.conn = nil
	return io.EOF
}

func (tpc *twilioWebsocketStreamer) Streamer(c *gin.Context, connection *websocket.Conn, assistantID uint64, assistantVersion string, assistantConversationID uint64) internal_streamers.Streamer {
	return NewTwilioWebsocketStreamer(tpc.logger, connection, assistantID,
		assistantVersion,
		assistantConversationID)
}
