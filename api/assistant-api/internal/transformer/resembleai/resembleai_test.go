package internal_transformer_resembleai

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResembleAITextToSpeechWaitsForTextClosureAndFinalDrain(t *testing.T) {
	upgrader := websocket.Upgrader{}
	requests := make(chan map[string]interface{}, 4)
	serverConn := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		serverConn <- conn
		for {
			var payload map[string]interface{}
			if err := conn.ReadJSON(&payload); err != nil {
				return
			}
			requests <- payload
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	remote := waitResembleAITTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &resembleaiTTS{
		resembleaiOption: &resembleaiOption{mdlOpts: utils.Option{}},
		ctx:              ctx,
		ctxCancel:        cancel,
		connection:       conn,
		contextId:        "ctx-resemble",
		logger:           testutil.NewTestLogger(),
		onPacket:         collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-resemble",
		Text:      "first",
	}))
	assert.Equal(t, "first", waitResembleAITTSRequest(t, requests)["data"])
	writeResembleAITTSAudio(t, remote, []byte{1})
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "audio_end"}))
	collector.WaitFor(t, time.Second, "first audio", func() bool {
		return len(collector.AudioPackets()) == 1
	})
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, collector.EndPackets())

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-resemble",
		Text:      "second",
	}))
	assert.Equal(t, "second", waitResembleAITTSRequest(t, requests)["data"])
	writeResembleAITTSAudio(t, remote, []byte{2})
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "audio_end"}))
	collector.WaitFor(t, time.Second, "second audio", func() bool {
		return len(collector.AudioPackets()) == 2
	})
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, collector.EndPackets())

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-resemble",
	}))
	collector.WaitForTTSEnd(t, time.Second)
	assert.Len(t, collector.AudioPackets(), 2)
	assert.Len(t, collector.EndPackets(), 1)
}

func TestResembleAITextToSpeechProviderErrorDoesNotEmitEnd(t *testing.T) {
	upgrader := websocket.Upgrader{}
	serverConn := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		serverConn <- conn
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	remote := waitResembleAITTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &resembleaiTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-resemble-error",
		logger:     testutil.NewTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, remote.WriteJSON(map[string]interface{}{
		"type":    "error",
		"message": "provider rejected request",
	}))
	collector.WaitFor(t, time.Second, "TTS error packet", func() bool {
		for _, packet := range collector.GetPackets() {
			if _, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
				return true
			}
		}
		return false
	})
	assert.Empty(t, collector.EndPackets())
}

func waitResembleAITTSServerConnection(t *testing.T, ch <-chan *websocket.Conn) *websocket.Conn {
	t.Helper()
	select {
	case conn := <-ch:
		return conn
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket connection")
		return nil
	}
}

func waitResembleAITTSRequest(t *testing.T, ch <-chan map[string]interface{}) map[string]interface{} {
	t.Helper()
	select {
	case request := <-ch:
		return request
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket request")
		return nil
	}
}

func writeResembleAITTSAudio(t *testing.T, conn *websocket.Conn, audio []byte) {
	t.Helper()
	require.NoError(t, conn.WriteJSON(map[string]interface{}{
		"type":          "audio",
		"audio_content": base64.StdEncoding.EncodeToString(audio),
	}))
}
