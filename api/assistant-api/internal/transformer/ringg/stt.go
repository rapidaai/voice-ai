// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_ringg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type ringgSTT struct {
	*ringgOption

	ctx       context.Context
	ctxCancel context.CancelFunc

	mu         sync.Mutex
	connectMu  sync.Mutex
	writeMu    sync.Mutex
	connection *websocket.Conn

	contextId      string
	sttConnectedAt time.Time
	startedAt      time.Time

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error

	dialWS func(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

type ringgWSResponse struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Language  string `json:"language,omitempty"`

	Transcription string `json:"transcription,omitempty"`
	IsFinal       bool   `json:"is_final,omitempty"`

	Detail     string `json:"detail,omitempty"`
	Code       string `json:"code,omitempty"`
	StatusCode *int   `json:"status_code,omitempty"`
}

func NewRinggSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.SpeechToTextTransformer, error) {
	ringgOpts, err := NewRinggOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("ringg-stt: initializing ringg failed %+v", err)
		return nil, err
	}

	ctx2, contextCancel := context.WithCancel(ctx)
	return &ringgSTT{
		ctx:         ctx2,
		ctxCancel:   contextCancel,
		onPacket:    onPacket,
		logger:      logger,
		ringgOption: ringgOpts,
		dialWS:      websocket.DefaultDialer.DialContext,
	}, nil
}

func (*ringgSTT) Name() string {
	return "ringg-speech-to-text"
}

func (st *ringgSTT) Initialize() error {
	start := time.Now()
	if _, err := st.getOrOpenConnection(); err != nil {
		return fmt.Errorf("ringg-stt: failed to connect: %w", err)
	}

	st.onPacket(internal_type.ConversationEventPacket{
		ContextID: st.currentContextID(),
		Name:      "stt",
		Data: map[string]string{
			"type":     "initialized",
			"provider": st.Name(),
			"init_ms":  fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		},
		Time: time.Now(),
	})
	return nil
}

func (st *ringgSTT) Transform(_ context.Context, in internal_type.Packet) error {
	switch pkt := in.(type) {
	case internal_type.TurnChangePacket:
		st.mu.Lock()
		st.contextId = pkt.ContextID
		st.mu.Unlock()
		return nil
	case internal_type.SpeechToTextEndPacket:
		st.mu.Lock()
		if st.startedAt.IsZero() {
			st.startedAt = time.Now()
		}
		if pkt.ContextID != "" {
			st.contextId = pkt.ContextID
		}
		st.mu.Unlock()
		return nil
	case internal_type.SpeechToTextAudioPacket:
		if len(pkt.Audio) == 0 {
			return nil
		}
		return st.handleAudio(pkt.ContextID, pkt.Audio)
	default:
		return nil
	}
}

func (st *ringgSTT) Close(_ context.Context) error {
	st.ctxCancel()
	st.connectMu.Lock()
	defer st.connectMu.Unlock()

	st.mu.Lock()
	conn := st.connection
	contextID := st.contextId
	connectedAt := st.sttConnectedAt
	st.connection = nil
	st.contextId = ""
	st.sttConnectedAt = time.Time{}
	st.startedAt = time.Time{}
	st.mu.Unlock()

	if conn != nil {
		st.writeMu.Lock()
		_ = conn.WriteJSON(map[string]string{"type": "end"})
		st.writeMu.Unlock()
		_ = conn.Close()
	}

	if !connectedAt.IsZero() {
		st.onPacket(
			internal_type.ConversationEventPacket{
				ContextID: contextID,
				Name:      "stt",
				Data: map[string]string{
					"type":     "closed",
					"provider": st.Name(),
				},
				Time: time.Now(),
			},
			internal_type.ConversationMetricPacket{
				Metrics: []*protos.Metric{{
					Name:        type_enums.CONVERSATION_STT_DURATION.String(),
					Value:       fmt.Sprintf("%d", time.Since(connectedAt).Nanoseconds()),
					Description: "Total STT connection duration in nanoseconds",
				}},
			},
		)
	}

	return nil
}

func (st *ringgSTT) handleAudio(contextID string, audio []byte) error {
	st.mu.Lock()
	if contextID != "" {
		st.contextId = contextID
	}
	effectiveContextID := st.contextId
	conn := st.connection
	st.mu.Unlock()

	if conn == nil {
		return nil
	}

	st.writeMu.Lock()
	err := conn.WriteMessage(websocket.BinaryMessage, audio)
	st.writeMu.Unlock()
	if err != nil {
		st.onPacket(internal_type.SpeechToTextErrorPacket{
			ContextID: effectiveContextID,
			Error:     fmt.Errorf("ringg-stt: send failed: %w", err),
			Type:      internal_type.STTNetworkTimeout,
		})
	}
	return nil
}

func (st *ringgSTT) getOrOpenConnection() (*websocket.Conn, error) {
	st.connectMu.Lock()
	defer st.connectMu.Unlock()

	st.mu.Lock()
	if st.connection != nil {
		conn := st.connection
		st.mu.Unlock()
		return conn, nil
	}
	st.mu.Unlock()

	dialWS := st.dialWS
	if dialWS == nil {
		dialWS = websocket.DefaultDialer.DialContext
	}

	conn, response, err := dialWS(st.ctx, RINGG_STT_URL, http.Header{})
	if err != nil {
		if response != nil {
			return nil, fmt.Errorf("ringg-stt: websocket dial failed: %w (status=%s)", err, response.Status)
		}
		return nil, fmt.Errorf("ringg-stt: websocket dial failed: %w", err)
	}

	cleanup := true
	defer func() {
		if cleanup {
			_ = conn.Close()
		}
	}()

	startPayload := map[string]any{
		"type":                     "start",
		"sample_rate":              16000,
		"encoding":                 "int16",
		"language":                 st.GetLanguage(),
		"mode":                     "stream",
		"vad_tail_sil_ms":          200,
		"vad_confidence":           0.55,
		"enable_cap_punc":          true,
		"accept_client_vad_events": false,
		"api_key":                  st.GetKey(),
	}

	st.writeMu.Lock()
	err = conn.WriteJSON(startPayload)
	st.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("ringg-stt: failed sending start request: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, fmt.Errorf("ringg-stt: failed setting handshake deadline: %w", err)
	}
	_, msg, err := conn.ReadMessage()
	_ = conn.SetReadDeadline(time.Time{})
	if err != nil {
		return nil, fmt.Errorf("ringg-stt: failed waiting for ready event: %w", err)
	}

	event, err := parseRinggEvent(msg)
	if err != nil {
		return nil, fmt.Errorf("ringg-stt: failed parsing ready event: %w", err)
	}
	if event.Type == "error" {
		return nil, fmt.Errorf("ringg-stt: server error during init: %s", ringgErrorMessage(event))
	}
	if event.Type != "ready" {
		return nil, fmt.Errorf("ringg-stt: expected ready event, got %q", event.Type)
	}

	st.mu.Lock()
	st.connection = conn
	st.sttConnectedAt = time.Now()
	st.mu.Unlock()

	cleanup = false
	go st.readLoop(conn)
	return conn, nil
}

func (st *ringgSTT) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-st.ctx.Done():
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			st.mu.Lock()
			intentional := st.connection == nil
			if !intentional {
				st.connection = nil
			}
			contextID := st.contextId
			st.mu.Unlock()
			if !intentional && !isContextClosedError(err) {
				st.onPacket(internal_type.SpeechToTextErrorPacket{
					ContextID: contextID,
					Error:     fmt.Errorf("ringg-stt: connection lost: %w", err),
					Type:      internal_type.STTNetworkTimeout,
				})
			}
			return
		}

		event, err := parseRinggEvent(msg)
		if err != nil {
			st.logger.Errorf("ringg-stt: error parsing response: %v", err)
			continue
		}

		st.mu.Lock()
		contextID := st.contextId
		st.mu.Unlock()

		switch event.Type {
		case "ready", "ack", "pong":
			continue
		case "error":
			st.onPacket(internal_type.SpeechToTextErrorPacket{
				ContextID: contextID,
				Error:     fmt.Errorf("ringg-stt: server error: %s", ringgErrorMessage(event)),
				Type:      ringgErrorType(event),
			})
			return
		case "transcript":
			transcript := strings.TrimSpace(event.Transcription)
			if transcript == "" {
				continue
			}

			language := event.Language
			if strings.TrimSpace(language) == "" {
				language = st.GetLanguage()
			}

			if !event.IsFinal {
				st.onPacket(
					internal_type.InterruptionDetectedPacket{
						ContextID: contextID,
						Source:    internal_type.InterruptionSourceWord,
					},
					internal_type.SpeechToTextPacket{
						ContextID: contextID,
						Script:    transcript,
						Language:  language,
						Interim:   true,
					},
					internal_type.ConversationEventPacket{
						ContextID: contextID,
						Name:      "stt",
						Data: map[string]string{
							"type":       "interim",
							"script":     transcript,
							"language":   language,
							"provider":   st.Name(),
							"request_id": event.RequestID,
						},
						Time: time.Now(),
					},
				)
				continue
			}

			now := time.Now()
			latencyMs := st.swapStartedAt(now)
			st.onPacket(
				internal_type.InterruptionDetectedPacket{
					ContextID: contextID,
					Source:    internal_type.InterruptionSourceWord,
				},
				internal_type.SpeechToTextPacket{
					ContextID: contextID,
					Script:    transcript,
					Language:  language,
					Interim:   false,
				},
				internal_type.ConversationEventPacket{
					ContextID: contextID,
					Name:      "stt",
					Data: map[string]string{
						"type":       "completed",
						"script":     transcript,
						"language":   language,
						"provider":   st.Name(),
						"request_id": event.RequestID,
					},
					Time: now,
				},
				internal_type.UserMessageMetricPacket{
					ContextID: contextID,
					Metrics: []*protos.Metric{{
						Name:        "stt_latency_ms",
						Value:       fmt.Sprintf("%d", latencyMs),
						Description: "Latency in milliseconds for speech-to-text processing",
					}},
				},
			)
		default:
			st.logger.Debugf("ringg-stt: unhandled message type: %s", event.Type)
		}
	}
}

func (st *ringgSTT) swapStartedAt(now time.Time) int64 {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.startedAt.IsZero() {
		return 0
	}
	latencyMs := now.Sub(st.startedAt).Milliseconds()
	st.startedAt = time.Time{}
	return latencyMs
}

func (st *ringgSTT) currentContextID() string {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.contextId
}

func parseRinggEvent(msg []byte) (*ringgWSResponse, error) {
	var event ringgWSResponse
	if err := json.Unmarshal(msg, &event); err != nil {
		return nil, err
	}
	if event.Type == "" {
		return nil, fmt.Errorf("missing type")
	}
	return &event, nil
}

func ringgErrorMessage(event *ringgWSResponse) string {
	if event == nil {
		return "unknown error"
	}
	if strings.TrimSpace(event.Detail) != "" {
		return event.Detail
	}
	if strings.TrimSpace(event.Code) != "" {
		return event.Code
	}
	return "unknown error"
}

func ringgErrorType(event *ringgWSResponse) internal_type.STTErrorType {
	if event != nil && event.StatusCode != nil {
		switch *event.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return internal_type.STTAuthentication
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			return internal_type.STTInvalidInput
		}
	}

	if event == nil {
		return internal_type.STTSystemPanic
	}

	code := strings.ToUpper(event.Code)
	switch {
	case strings.Contains(code, "AUTH"), strings.Contains(code, "TOKEN"), strings.Contains(code, "KEY"):
		return internal_type.STTAuthentication
	case strings.Contains(code, "INVALID"), strings.Contains(code, "BAD_REQUEST"), strings.Contains(code, "UNSUPPORTED"):
		return internal_type.STTInvalidInput
	default:
		return internal_type.STTSystemPanic
	}
}

func isContextClosedError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || strings.Contains(strings.ToLower(err.Error()), "use of closed network connection")
}
