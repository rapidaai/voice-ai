package internal_adapter_request_streamers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/gorilla/websocket"
	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/commons"
	lexatic_backend "github.com/rapidaai/protos"
)

type exotelWebsocketStreamer struct {
	logger     commons.Logger
	conn       *websocket.Conn
	ctx        context.Context
	cancelFunc context.CancelFunc

	assistant               *lexatic_backend.AssistantDefinition
	version                 string
	assistantConversationId uint64
	streamSid               string
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
) Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	return &exotelWebsocketStreamer{
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
	}
}

func (exotel *exotelWebsocketStreamer) Context() context.Context {
	return exotel.ctx
}

func (exotel *exotelWebsocketStreamer) Recv() (*lexatic_backend.AssistantMessagingRequest, error) {
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
		payloadBytes, err := base64.StdEncoding.DecodeString(mediaEvent.Media.Payload)
		if err != nil {
			exotel.logger.Warn("Failed to decode media payload", "error", err.Error())
			return nil, nil
		}

		request := &lexatic_backend.AssistantMessagingRequest{
			Request: &lexatic_backend.AssistantMessagingRequest_Message{
				Message: &lexatic_backend.AssistantConversationUserMessage{
					Message: &lexatic_backend.AssistantConversationUserMessage_Audio{
						Audio: &lexatic_backend.AssistantConversationMessageAudioContent{
							Content: payloadBytes,
						},
					},
				},
			},
		}
		return request, nil

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

func (exotel *exotelWebsocketStreamer) Send(response *lexatic_backend.AssistantMessagingResponse) error {
	if response.GetMessage() == nil || exotel.conn == nil {
		return nil
	}
	if exotel.streamSid == "" {
		exotel.logger.Warn("StreamSid is empty, cannot send message")
		return nil
	}

	switch response.GetData().(type) {
	case *lexatic_backend.AssistantMessagingResponse_Message:
		for _, content := range response.GetMessage().GetResponse().GetContents() {
			twilioMessageJSON, err := json.Marshal(map[string]interface{}{
				"event":      "media",
				"stream_sid": exotel.streamSid,
				"media": map[string]interface{}{
					"payload": base64.StdEncoding.EncodeToString(content.GetContent()),
				},
			})
			if err != nil {
				exotel.logger.Error("Failed to marshal Twilio message", "error", err.Error())
				return err
			}

			err = exotel.conn.WriteMessage(websocket.TextMessage, twilioMessageJSON)
			if err != nil {
				exotel.logger.Error("Failed to send message to Twilio", "error", err.Error())
				return err
			}
		}
	case *lexatic_backend.AssistantMessagingResponse_Interruption:
		exotelClearJson, err := json.Marshal(map[string]interface{}{
			"event":     "clear",
			"streamSid": exotel.streamSid,
		})
		if err != nil {
			exotel.logger.Error("Failed to marshal Twilio message", "error", err.Error())
			return err
		}
		err = exotel.conn.WriteMessage(websocket.TextMessage, exotelClearJson)
		if err != nil {
			exotel.logger.Error("Failed to send clear event to Twilio", "error", err.Error())
			return err
		}

	}

	return nil
}

func (extl *exotelWebsocketStreamer) Config() *StreamAttribute {
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
