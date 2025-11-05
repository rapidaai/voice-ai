package internal_adapter_request_streamers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"sync"

	"github.com/gorilla/websocket"
	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/commons"
	lexatic_backend "github.com/rapidaai/protos"
)

type twilioWebsocketStreamer struct {
	logger     commons.Logger
	conn       *websocket.Conn
	ctx        context.Context
	cancelFunc context.CancelFunc

	assistant               *lexatic_backend.AssistantDefinition
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
) Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	return &twilioWebsocketStreamer{
		logger:     logger,
		conn:       connection,
		ctx:        ctx,
		cancelFunc: cancel,
		assistant: &lexatic_backend.AssistantDefinition{
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

func (tws *twilioWebsocketStreamer) Recv() (*lexatic_backend.AssistantMessagingRequest, error) {
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
	tws.captureStreamSid(mediaEvent.StreamSid)
	switch mediaEvent.Event {
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

func (tws *twilioWebsocketStreamer) handleMediaEvent(mediaEvent TwilioMediaEvent) (*lexatic_backend.AssistantMessagingRequest, error) {
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
		audioRequest := tws.BuildVoiceRequest(tws.inputAudioBuffer.Bytes())
		tws.inputAudioBuffer.Reset()
		return audioRequest, nil
	}

	return nil, nil
}

func (tws *twilioWebsocketStreamer) BuildVoiceRequest(audioData []byte) *lexatic_backend.AssistantMessagingRequest {
	return &lexatic_backend.AssistantMessagingRequest{
		Request: &lexatic_backend.AssistantMessagingRequest_Message{
			Message: &lexatic_backend.AssistantConversationUserMessage{
				Message: &lexatic_backend.AssistantConversationUserMessage_Audio{
					Audio: &lexatic_backend.AssistantConversationMessageAudioContent{
						Content: audioData,
					},
				},
			},
		},
	}
}

func (tws *twilioWebsocketStreamer) Send(response *lexatic_backend.AssistantMessagingResponse) error {

	switch data := response.GetData().(type) {
	case *lexatic_backend.AssistantMessagingResponse_Assistant:
		switch content := data.Assistant.Message.(type) {
		case *lexatic_backend.AssistantConversationAssistantMessage_Audio:
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
	case *lexatic_backend.AssistantMessagingResponse_Interruption:
		tws.logger.Debugf("clearing action")
		tws.audioBufferLock.Lock()
		defer tws.audioBufferLock.Unlock()
		tws.outputAudioBuffer.Reset() // Clear the buffer after flushing
		err := tws.sendTwilioMessage("clear", nil)
		if err != nil {
			tws.logger.Errorf("Error sending clear command:", err)
		}
	case *lexatic_backend.AssistantMessagingResponse_DisconnectAction:
		tws.logger.Debugf("ending call action")
		err := tws.conn.Close()
		if err != nil {
			tws.logger.Errorf("Error disconnecting command:", err)
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

func (tws *twilioWebsocketStreamer) captureStreamSid(streamSid string) {
	if tws.streamSid == "" && streamSid != "" {
		tws.streamSid = streamSid
		tws.logger.Debug("Captured Twilio streamSid", "streamSid", tws.streamSid)
	}
}

func (uds *twilioWebsocketStreamer) Config() *StreamAttribute {
	return &StreamAttribute{
		InputConfig: &StreamConfig{
			Audio: internal_voices.NewMulaw8khzMonoAudioConfig(),
			Text: &struct {
				Charset string `json:"charset"`
			}{
				Charset: "UTF-8",
			},
		},
		OutputConfig: &StreamConfig{
			Audio: internal_voices.NewMulaw8khzMonoAudioConfig(),
			Text: &struct {
				Charset string `json:"charset"`
			}{
				Charset: "UTF-8",
			},
		},
	}
}
