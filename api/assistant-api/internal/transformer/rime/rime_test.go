package internal_transformer_rime

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
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestLogger() commons.Logger {
	l, _ := commons.NewApplicationLogger()
	return l
}

func newVaultCredential(m map[string]interface{}) *protos.VaultCredential {
	val, _ := structpb.NewStruct(m)
	return &protos.VaultCredential{Value: val}
}

// --- Constructor Tests ---

func TestNewRimeOption_ValidCredentials(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "test-api-key"})
	opt, err := NewRimeOption(newTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "test-api-key", opt.GetKey())
}

func TestNewRimeOption_MissingKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"other": "value"})
	opt, err := NewRimeOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "missing 'key'")
}

func TestNewRimeOption_EmptyVault(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{})
	opt, err := NewRimeOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
}

// --- Connection String Tests ---

func TestGetTextToSpeechConnectionString_Default(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewRimeOption(newTestLogger(), cred, utils.Option{})
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "wss://users-ws.rime.ai/ws3?")
	assert.Contains(t, connStr, "audioFormat=pcm")
	assert.Contains(t, connStr, "samplingRate=16000")
	assert.Contains(t, connStr, "segment=never")
	assert.Contains(t, connStr, "speaker="+RIME_DEFAULT_VOICE)
	assert.Contains(t, connStr, "modelId="+RIME_DEFAULT_MODEL)
	assert.Contains(t, connStr, "lang="+RIME_DEFAULT_LANG)
	assert.NotContains(t, connStr, "speedAlpha=")
}

func TestGetTextToSpeechConnectionString_WithOverrides(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id":    "aria",
		"speak.model":       "mist",
		"speak.language":    "fra",
		"speak.speed_alpha": "1.2",
	}
	opt, _ := NewRimeOption(newTestLogger(), cred, opts)
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "speaker=aria")
	assert.Contains(t, connStr, "modelId=mist")
	assert.Contains(t, connStr, "lang=fra")
	assert.Contains(t, connStr, "speedAlpha=1.2")
	assert.Contains(t, connStr, "audioFormat=pcm")
	assert.Contains(t, connStr, "samplingRate=16000")
}

func TestGetTextToSpeechConnectionString_PartialOverrides(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "custom-voice",
	}
	opt, _ := NewRimeOption(newTestLogger(), cred, opts)
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "speaker=custom-voice")
	assert.Contains(t, connStr, "modelId="+RIME_DEFAULT_MODEL)
	assert.Contains(t, connStr, "lang="+RIME_DEFAULT_LANG)
}

func TestRimeTextToSpeechWaitsForTextClosureAndFinalDrain(t *testing.T) {
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
	remote := waitRimeTTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &rimeTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-rime",
		logger:     newTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-rime",
		Text:      "first",
	}))
	assert.Equal(t, "first", waitRimeTTSRequest(t, requests)["text"])
	writeRimeTTSChunk(t, remote, []byte{1})
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "contextId": "ctx-rime"}))
	collector.WaitFor(t, time.Second, "first audio", func() bool {
		return len(collector.AudioPackets()) == 1
	})
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, collector.EndPackets())

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-rime",
		Text:      "second",
	}))
	assert.Equal(t, "second", waitRimeTTSRequest(t, requests)["text"])
	writeRimeTTSChunk(t, remote, []byte{2})
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "contextId": "ctx-rime"}))
	collector.WaitFor(t, time.Second, "second audio", func() bool {
		return len(collector.AudioPackets()) == 2
	})
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, collector.EndPackets())

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-rime",
	}))
	assert.Equal(t, "eos", waitRimeTTSRequest(t, requests)["operation"])
	writeRimeTTSChunk(t, remote, []byte{3})
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "contextId": "ctx-rime"}))
	require.NoError(t, remote.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	))
	collector.WaitForTTSEnd(t, time.Second)
	assert.Len(t, collector.AudioPackets(), 3)
	assert.Len(t, collector.EndPackets(), 1)
}

func TestRimeTextToSpeechLatePreFinalDoneAfterEOSWaitsForRemoteClose(t *testing.T) {
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
	remote := waitRimeTTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &rimeTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-rime-late",
		logger:     newTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-rime-late",
		Text:      "first",
	}))
	assert.Equal(t, "first", waitRimeTTSRequest(t, requests)["text"])
	require.NoError(t, remote.WriteJSON(map[string]interface{}{
		"type":      "chunk",
		"data":      base64.StdEncoding.EncodeToString([]byte{1}),
		"contextId": "ctx-rime-late",
	}))
	collector.WaitFor(t, time.Second, "first audio", func() bool {
		return len(collector.AudioPackets()) == 1
	})

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-rime-late",
	}))
	assert.Equal(t, "eos", waitRimeTTSRequest(t, requests)["operation"])
	require.NoError(t, remote.WriteJSON(map[string]interface{}{"type": "done", "contextId": "ctx-rime-late"}))
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, collector.EndPackets())

	require.NoError(t, remote.WriteJSON(map[string]interface{}{
		"type":      "chunk",
		"data":      base64.StdEncoding.EncodeToString([]byte{2}),
		"contextId": "ctx-rime-late",
	}))
	require.NoError(t, remote.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	))
	collector.WaitForTTSEnd(t, time.Second)
	packets := collector.GetPackets()
	var lastAudioIndex, endIndex = -1, -1
	for i, packet := range packets {
		switch packet.(type) {
		case internal_type.TextToSpeechAudioPacket:
			lastAudioIndex = i
		case internal_type.TextToSpeechEndPacket:
			endIndex = i
		}
	}
	require.NotEqual(t, -1, lastAudioIndex)
	require.NotEqual(t, -1, endIndex)
	assert.Less(t, lastAudioIndex, endIndex)
	assert.Len(t, collector.AudioPackets(), 2)
	assert.Len(t, collector.EndPackets(), 1)
}

func TestRimeTextToSpeechZeroBufferEOSCompletesOnCleanRemoteClose(t *testing.T) {
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
	remote := waitRimeTTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &rimeTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-rime-empty",
		logger:     newTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-rime-empty",
	}))
	assert.Equal(t, "eos", waitRimeTTSRequest(t, requests)["operation"])
	require.NoError(t, remote.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	))
	collector.WaitForTTSEnd(t, time.Second)
	assert.Empty(t, collector.AudioPackets())
	assert.Len(t, collector.EndPackets(), 1)
}

func TestRimeTextToSpeechProviderErrorDoesNotEmitEnd(t *testing.T) {
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
	remote := waitRimeTTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &rimeTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-rime-error",
		logger:     newTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, remote.WriteJSON(map[string]interface{}{
		"type":      "error",
		"message":   "provider rejected request",
		"contextId": "ctx-rime-error",
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

func TestRimeTextToSpeechLocalCloseAfterEOSDoesNotEmitEnd(t *testing.T) {
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
	remote := waitRimeTTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &rimeTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-rime-local-close",
		logger:     newTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-rime-local-close",
	}))
	assert.Equal(t, "eos", waitRimeTTSRequest(t, requests)["operation"])
	require.NoError(t, tts.Close(context.Background()))
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, collector.EndPackets())
}

func TestRimeShutdownCloseDoesNotCompleteSynthesis(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		closeCode int
		cancelled bool
	}{
		{name: "server going away", closeCode: websocket.CloseGoingAway},
		{name: "cancelled context", closeCode: websocket.CloseNormalClosure, cancelled: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			upgrader := websocket.Upgrader{}
			serverConnection := make(chan *websocket.Conn, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				connection, err := upgrader.Upgrade(w, r, nil)
				if err == nil {
					serverConnection <- connection
				}
			}))
			defer server.Close()
			connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err)
			defer connection.Close()
			remote := waitRimeTTSServerConnection(t, serverConnection)
			defer remote.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			collector := testutil.NewPacketCollector()
			tts := &rimeTTS{ctx: ctx, connection: connection, contextId: "closing", textClosed: true, logger: newTestLogger(), onPacket: collector.OnPacket}
			finished := make(chan struct{})
			go func() { defer close(finished); tts.readLoop(connection) }()
			if scenario.cancelled {
				cancel()
			}
			require.NoError(t, remote.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(scenario.closeCode, ""), time.Now().Add(time.Second)))
			select {
			case <-finished:
			case <-time.After(time.Second):
				t.Fatal("read loop did not close")
			}
			assert.Empty(t, collector.EndPackets())
		})
	}
}

func waitRimeTTSServerConnection(t *testing.T, ch <-chan *websocket.Conn) *websocket.Conn {
	t.Helper()
	select {
	case conn := <-ch:
		return conn
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket connection")
		return nil
	}
}

func waitRimeTTSRequest(t *testing.T, ch <-chan map[string]interface{}) map[string]interface{} {
	t.Helper()
	select {
	case request := <-ch:
		return request
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket request")
		return nil
	}
}

func writeRimeTTSChunk(t *testing.T, conn *websocket.Conn, audio []byte) {
	t.Helper()
	require.NoError(t, conn.WriteJSON(map[string]interface{}{
		"type":      "chunk",
		"data":      base64.StdEncoding.EncodeToString(audio),
		"contextId": "ctx-rime",
	}))
}
