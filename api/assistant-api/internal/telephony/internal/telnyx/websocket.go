// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_telnyx_telephony

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gorilla/websocket"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/telephony/internal/base"
	internal_telnyx "github.com/rapidaai/api/assistant-api/internal/telephony/internal/telnyx/internal"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type telnyxWebsocketStreamer struct {
	streamID string
	streamer internal_telephony_base.BaseTelephonyStreamer
	logger   commons.Logger
}

func NewTelnyxWebsocketStreamer(logger commons.Logger, connection *websocket.Conn, assistant *internal_assistant_entity.Assistant, conversation *internal_conversation_entity.AssistantConversation, vlt *protos.VaultCredential) internal_streamers.Streamer {
	return &telnyxWebsocketStreamer{
		logger:   logger,
		streamID: "",
		streamer: internal_telephony_base.NewBaseTelephonyStreamer(logger, connection, assistant, conversation, vlt),
	}
}

func (tws *telnyxWebsocketStreamer) Context() context.Context {
	return tws.streamer.Context()
}

func (tws *telnyxWebsocketStreamer) Recv() (*protos.AssistantTalkInput, error) {
	if tws.streamer.Connection() == nil {
		return nil, tws.handleError("WebSocket connection is nil", io.EOF)
	}
	_, message, err := tws.streamer.Connection().ReadMessage()
	if err != nil {
		return nil, tws.handleWebSocketError(err)
	}

	var mediaEvent internal_telnyx.TelnyxMediaEvent
	if err := json.Unmarshal(message, &mediaEvent); err != nil {
		tws.logger.Error("Failed to unmarshal Telnyx media event", "error", err.Error())
		return nil, nil
	}
	switch mediaEvent.Event {
	case "connected":
		return tws.streamer.CreateConnectionRequest(internal_audio.NewLinear16khzMonoAudioConfig(), internal_audio.NewLinear16khzMonoAudioConfig())
	case "start":
		tws.handleStartEvent(mediaEvent)
		return nil, nil
	case "media":
		return tws.handleMediaEvent(mediaEvent)
	case "stop":
		tws.logger.Info("Telnyx stream stopped")
		tws.streamer.Cancel()
		return nil, io.EOF
	default:
		tws.logger.Warn("Unhandled Telnyx event", "event", mediaEvent.Event)
		return nil, nil
	}
}

func (tws *telnyxWebsocketStreamer) Send(response *protos.AssistantTalkOutput) error {
	switch data := response.GetData().(type) {
	case *protos.AssistantTalkOutput_Assistant:
		switch content := data.Assistant.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			// 1ms 32 bytes @ 16000Hz, 16-bit mono PCM = 32 bytes
			// Each message needs to be a 20ms sample of audio.
			// At 16kHz the message should be 640 bytes.
			bufferSizeThreshold := 32 * 20
			audioData := content.Audio

			// Use audioBuffer to handle pending data across calls
			tws.streamer.LockOutputAudioBuffer()
			defer tws.streamer.UnlockOutputAudioBuffer()

			// Append incoming audio data to the buffer
			tws.streamer.OutputBuffer().Write(audioData)
			// Process and send chunks of 640 bytes
			for tws.streamer.OutputBuffer().Len() >= bufferSizeThreshold && tws.streamID != "" {
				chunk := tws.streamer.OutputBuffer().Next(bufferSizeThreshold)
				if err := tws.sendTelnyxMessage("media", map[string]interface{}{
					"payload": chunk,
				}); err != nil {
					tws.logger.Error("Failed to send audio chunk", "error", err.Error())
					return err
				}
			}

			// If response is marked as completed, flush any remaining audio in the buffer
			if data.Assistant.GetCompleted() && tws.streamer.OutputBuffer().Len() > 0 {
				remainingChunk := tws.streamer.OutputBuffer().Bytes()
				if err := tws.sendTelnyxMessage("media", map[string]interface{}{
					"payload": remainingChunk,
				}); err != nil {
					tws.logger.Errorf("Failed to send final audio chunk", "error", err.Error())
					return err
				}
				tws.streamer.OutputBuffer().Reset()
			}
		}
	case *protos.AssistantTalkOutput_Interruption:
		if data.Interruption.Type == protos.ConversationInterruption_INTERRUPTION_TYPE_WORD {
			tws.streamer.LockOutputAudioBuffer()
			tws.streamer.OutputBuffer().Reset()
			tws.streamer.UnlockOutputAudioBuffer()

			if err := tws.sendTelnyxMessage("clear", nil); err != nil {
				tws.logger.Errorf("Error sending clear command:", err)
			}
		}
	case *protos.AssistantTalkOutput_Directive:
		if data.Directive.GetType() == protos.ConversationDirective_END_CONVERSATION {
			if tws.streamer.GetUuid() != "" {
				if err := tws.endCall(); err != nil {
					tws.logger.Errorf("Error ending Telnyx call:", err)
				}
			}
			if err := tws.streamer.Cancel(); err != nil {
				tws.logger.Errorf("Error disconnecting command:", err)
			}
		}
	}
	return nil
}

// start event contains streamSid to be used for subsequent media messages
func (tws *telnyxWebsocketStreamer) handleStartEvent(mediaEvent internal_telnyx.TelnyxMediaEvent) {
	tws.streamID = mediaEvent.StreamSid
}

func (tws *telnyxWebsocketStreamer) handleMediaEvent(mediaEvent internal_telnyx.TelnyxMediaEvent) (*protos.AssistantTalkInput, error) {
	if mediaEvent.Media == nil || mediaEvent.Media.Payload == "" {
		return nil, nil
	}

	tws.streamer.LockInputAudioBuffer()
	defer tws.streamer.UnlockInputAudioBuffer()

	tws.streamer.InputBuffer().Write([]byte(mediaEvent.Media.Payload))
	const bufferSizeThreshold = 32 * 60

	if tws.streamer.InputBuffer().Len() >= bufferSizeThreshold {
		audioRequest := tws.streamer.CreateVoiceRequest(tws.streamer.InputBuffer().Bytes())
		tws.streamer.InputBuffer().Reset()
		return audioRequest, nil
	}

	return nil, nil
}

func (tws *telnyxWebsocketStreamer) sendTelnyxMessage(
	eventType string,
	mediaData map[string]interface{}) error {
	if tws.streamer.Connection() == nil || tws.streamID == "" {
		return nil
	}
	message := map[string]interface{}{
		"event":     eventType,
		"streamSid": tws.streamID,
	}
	if mediaData != nil {
		message["media"] = mediaData
	}

	telnyxMessageJSON, err := json.Marshal(message)
	if err != nil {
		return tws.handleError("Failed to marshal Telnyx message", err)
	}

	if err := tws.streamer.Connection().WriteMessage(websocket.TextMessage, telnyxMessageJSON); err != nil {
		return tws.handleError("Failed to send message to Telnyx", err)
	}

	return nil
}

func (tws *telnyxWebsocketStreamer) handleError(message string, err error) error {
	tws.logger.Error(message, "error", err.Error())
	return err
}

func (tws *telnyxWebsocketStreamer) handleWebSocketError(err error) error {
	if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
		tws.logger.Error("Unexpected websocket close error", "error", err.Error())
	} else {
		tws.logger.Error("Failed to read message from WebSocket", "error", err.Error())
	}
	tws.streamer.Cancel()
	return io.EOF
}

func (tws *telnyxWebsocketStreamer) endCall() error {
	return nil
}