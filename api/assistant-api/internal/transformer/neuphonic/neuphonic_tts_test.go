package internal_transformer_neuphonic

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func neuphonicTestLogger() commons.Logger {
	l, _ := commons.NewApplicationLogger()
	return l
}

func TestNeuphonicTextToSpeechAudioBeforeDoneFailure(t *testing.T) {
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
	remote := waitNeuphonicTTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &neuphonicTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-neuphonic-audio",
		logger:     neuphonicTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-neuphonic-audio",
		Text:      "hello",
	}))
	assert.Equal(t, "hello <STOP>", waitNeuphonicTTSRequest(t, requests)["text"])
	require.NoError(t, remote.WriteJSON(map[string]interface{}{
		"data": map[string]interface{}{
			"audio": base64.StdEncoding.EncodeToString([]byte{1, 2, 3}),
		},
	}))
	collector.WaitFor(t, time.Second, "neuphonic audio", func() bool {
		return len(collector.AudioPackets()) == 1
	})

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-neuphonic-audio",
	}))
	assert.Equal(t, "<STOP>", waitNeuphonicTTSRequest(t, requests)["text"])
	collector.WaitFor(t, time.Second, "neuphonic final error", func() bool {
		return len(neuphonicTTSErrors(collector)) == 1
	})
	assert.Empty(t, collector.EndPackets())
	assert.False(t, neuphonicHasTTSCompleted(collector))
}

func TestNeuphonicTextToSpeechFailedStopDoesNotComplete(t *testing.T) {
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
	remote := waitNeuphonicTTSServerConnection(t, serverConn)
	defer remote.Close()
	require.NoError(t, conn.Close())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &neuphonicTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-neuphonic-stop",
		logger:     neuphonicTestLogger(),
		onPacket:   collector.OnPacket,
	}

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-neuphonic-stop",
	}))
	errors := neuphonicTTSErrors(collector)
	require.Len(t, errors, 1)
	assert.Equal(t, internal_type.TTSNetworkTimeout, errors[0].Type)
	assert.Empty(t, collector.EndPackets())
	assert.False(t, neuphonicHasTTSCompleted(collector))
}

func TestNeuphonicTextToSpeechNoAudioDoneEmitsError(t *testing.T) {
	upgrader := websocket.Upgrader{}
	requests := make(chan map[string]interface{}, 2)
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
	remote := waitNeuphonicTTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &neuphonicTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-neuphonic-empty",
		logger:     neuphonicTestLogger(),
		onPacket:   collector.OnPacket,
	}

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-neuphonic-empty",
	}))
	assert.Equal(t, "<STOP>", waitNeuphonicTTSRequest(t, requests)["text"])
	errors := neuphonicTTSErrors(collector)
	require.Len(t, errors, 1)
	assert.Equal(t, internal_type.TTSNetworkTimeout, errors[0].Type)
	assert.Empty(t, collector.EndPackets())
	assert.False(t, neuphonicHasTTSCompleted(collector))
}

func TestNeuphonicTextToSpeechLocalCloseDoesNotComplete(t *testing.T) {
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
	remote := waitNeuphonicTTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &neuphonicTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-neuphonic-close",
		logger:     neuphonicTestLogger(),
		onPacket:   collector.OnPacket,
	}

	require.NoError(t, tts.Close(context.Background()))
	assert.Empty(t, neuphonicTTSErrors(collector))
	assert.Empty(t, collector.EndPackets())
	assert.False(t, neuphonicHasTTSCompleted(collector))
}

func waitNeuphonicTTSServerConnection(t *testing.T, ch <-chan *websocket.Conn) *websocket.Conn {
	t.Helper()
	select {
	case conn := <-ch:
		return conn
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket connection")
		return nil
	}
}

func waitNeuphonicTTSRequest(t *testing.T, ch <-chan map[string]interface{}) map[string]interface{} {
	t.Helper()
	select {
	case request := <-ch:
		return request
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket request")
		return nil
	}
}

func neuphonicTTSErrors(collector *testutil.PacketCollector) []internal_type.TextToSpeechErrorPacket {
	var errors []internal_type.TextToSpeechErrorPacket
	for _, packet := range collector.GetPackets() {
		if errorPacket, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			errors = append(errors, errorPacket)
		}
	}
	return errors
}

func neuphonicHasTTSCompleted(collector *testutil.PacketCollector) bool {
	for _, packet := range collector.EventPackets() {
		if packet.Record.Event == observability.TTSCompleted {
			return true
		}
	}
	return false
}
