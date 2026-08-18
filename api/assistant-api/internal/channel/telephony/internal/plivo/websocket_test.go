// Copyright (c) 2023-2025 RapidaAI
// Author: Sarvesh Patil <sarvesh.patil@plivo.com>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_plivo_telephony

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	internal_plivo "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/plivo/internal"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePlivoMediaEngine struct {
	providerFrame internal_telephony_media.ProviderAudioFrame
	processError  error
}

func (engine *fakePlivoMediaEngine) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
	engine.providerFrame = frame
	if engine.processError != nil {
		return internal_telephony_media.InputAudioFrame{}, engine.processError
	}
	return internal_telephony_media.InputAudioFrame{
		BridgeAudio:   []byte{1},
		PipelineAudio: []byte{2},
		ReceivedAt:    frame.ReceivedAt,
	}, nil
}

func (engine *fakePlivoMediaEngine) ProcessAssistantAudio(_ []byte, _ bool) error { return nil }

func (engine *fakePlivoMediaEngine) NextOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakePlivoMediaEngine) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	return internal_telephony_media.AssistantOutputFrame{}, false
}

func (engine *fakePlivoMediaEngine) ClearOutputBuffer() {}

func (engine *fakePlivoMediaEngine) ConfigureAmbient(_ internal_ambient.Config) error { return nil }

func (engine *fakePlivoMediaEngine) OutputFrameDuration() time.Duration { return 20 * time.Millisecond }

func (engine *fakePlivoMediaEngine) OutputHealthSnapshot() internal_output.HealthSnapshot {
	return internal_output.HealthSnapshot{}
}

func (engine *fakePlivoMediaEngine) OnTickHealth(_ internal_output.TickHealth) {}

func newTestCallContext() *callcontext.CallContext {
	return &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 2,
		Provider:       internal_plivo.PlivoProvider,
	}
}

func TestNewPlivoWebsocketStreamer_WiresMediaSession(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	streamer, err := New(
		WithLogger(logger),
		WithConnection(nil),
		WithCallContext(newTestCallContext()),
		WithVaultCredential(nil),
	)
	require.NoError(t, err)
	plivoStreamer, ok := streamer.(*plivoWebsocketStreamer)
	require.True(t, ok, "expected plivo websocket streamer")
	defer plivoStreamer.Cancel()

	require.NotNil(t, plivoStreamer.mediaSession)
}

func TestHandleStartEvent_AdoptsCallUUID(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	plivoStreamer := &plivoWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, newTestCallContext(), nil, nil),
	}

	plivoStreamer.handleStartEvent(internal_plivo.PlivoMediaEvent{
		Event:    internal_plivo.EventTypeStart,
		StreamID: "stream-1",
		Start: &internal_plivo.PlivoStart{
			CallID:   "call-uuid-abc",
			StreamID: "stream-1",
		},
	})

	if plivoStreamer.streamID != "stream-1" {
		t.Errorf("streamID=%s want stream-1", plivoStreamer.streamID)
	}
	if plivoStreamer.GetConversationUuid() != "call-uuid-abc" {
		t.Errorf("ChannelUUID=%s want call-uuid-abc", plivoStreamer.GetConversationUuid())
	}
}

func TestHandleMediaEvent_EmitsBridgeUserAudio(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	mediaEngine := &fakePlivoMediaEngine{}
	plivoStreamer := &plivoWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, newTestCallContext(), nil, nil),
	}
	plivoStreamer.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     plivoStreamer.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  plivoStreamer.Input,
	})

	providerAudio := []byte{9, 8, 7}
	mediaEvent := internal_plivo.PlivoMediaEvent{
		Event: internal_plivo.EventTypeMedia,
		Media: &internal_plivo.PlivoMedia{
			Payload: plivoStreamer.Encoder().EncodeToString(providerAudio),
		},
	}
	err := plivoStreamer.handleMediaEvent(mediaEvent)
	require.NoError(t, err)

	select {
	case stream := <-plivoStreamer.InputCh:
		bridgeAudio, ok := stream.(*protos.ConversationBridgeUserAudio)
		require.True(t, ok, "expected bridge user audio, got %T", stream)
		assert.NotEmpty(t, bridgeAudio.GetAudio())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bridge user audio")
	}
	assert.Equal(t, providerAudio, mediaEngine.providerFrame.Audio)
	assert.False(t, mediaEngine.providerFrame.ReceivedAt.IsZero())
}

func TestHandleMediaEvent_ReturnsMediaProcessingError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	mediaEngine := &fakePlivoMediaEngine{processError: errors.New("media process failed")}
	plivoStreamer := &plivoWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, newTestCallContext(), nil, nil),
	}
	plivoStreamer.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     plivoStreamer.Ctx,
		Logger:      logger,
		MediaEngine: mediaEngine,
		StreamSink:  plivoStreamer.Input,
	})

	mediaEvent := internal_plivo.PlivoMediaEvent{
		Media: &internal_plivo.PlivoMedia{
			Payload: plivoStreamer.Encoder().EncodeToString([]byte{9, 8, 7}),
		},
	}
	err := plivoStreamer.handleMediaEvent(mediaEvent)
	require.ErrorContains(t, err, "media process failed")
}

func TestHandleMediaEvent_MissingPayloadDoesNotPanic(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	plivoStreamer := &plivoWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, newTestCallContext(), nil, nil),
	}
	err := plivoStreamer.handleMediaEvent(internal_plivo.PlivoMediaEvent{})
	require.NoError(t, err)
}

func newBoundStreamer(t *testing.T, serverConn *websocket.Conn) *plivoWebsocketStreamer {
	t.Helper()
	logger, _ := commons.NewApplicationLogger()
	return &plivoWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, newTestCallContext(), nil, nil),
		connection:            serverConn,
	}
}

func readOutbound(t *testing.T, clientConn *websocket.Conn) internal_plivo.PlivoOutboundMessage {
	t.Helper()
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := clientConn.ReadMessage()
	require.NoError(t, err)
	var msg internal_plivo.PlivoOutboundMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	return msg
}

func TestOutboundMessages_RoundTrip(t *testing.T) {
	serverConnCh := make(chan *websocket.Conn, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		serverConnCh <- conn
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()
	serverConn := <-serverConnCh
	defer serverConn.Close()

	streamer := newBoundStreamer(t, serverConn)

	// Outbound messages must target the active stream via streamId.
	streamer.streamID = "MZtest0000000000000000000000000000"

	// playAudio carries base64 mu-law and, per Plivo's protocol, no streamId.
	require.NoError(t, streamer.sendOutputFrame(internal_telephony_media.AssistantOutputFrame{
		ProviderAudio: []byte{1, 2, 3, 4},
	}))
	playMsg := readOutbound(t, clientConn)
	assert.Equal(t, internal_plivo.EventTypePlayAudio, playMsg.Event)
	assert.Empty(t, playMsg.StreamID)
	require.NotNil(t, playMsg.Media)
	assert.Equal(t, internal_plivo.OutboundContentType, playMsg.Media.ContentType)
	assert.Equal(t, internal_plivo.OutboundSampleRate, playMsg.Media.SampleRate)
	assert.NotEmpty(t, playMsg.Media.Payload)

	// barge-in flushes with clearAudio, carrying the streamId (no media).
	require.NoError(t, streamer.sendClearAudio())
	clearMsg := readOutbound(t, clientConn)
	assert.Equal(t, internal_plivo.EventTypeClearAudio, clearMsg.Event)
	assert.Equal(t, streamer.streamID, clearMsg.StreamID)
	assert.Nil(t, clearMsg.Media)
}

func TestSendOutputFrame_EmptyAudioSendsNothing(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	streamer := &plivoWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, newTestCallContext(), nil, nil),
	}
	// No connection set; empty audio must be a no-op and not panic.
	require.NoError(t, streamer.sendOutputFrame(internal_telephony_media.AssistantOutputFrame{}))
}
