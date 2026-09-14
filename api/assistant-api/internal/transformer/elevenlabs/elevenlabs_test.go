package internal_transformer_elevenlabs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func newVaultCredential(m map[string]interface{}) *protos.VaultCredential {
	val, _ := structpb.NewStruct(m)
	return &protos.VaultCredential{Value: val}
}

// --- Constructor Tests ---

func TestNewElevenLabsOption_ValidCredentials(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "test-api-key"})
	opt, err := NewElevenLabsOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "test-api-key", opt.GetKey())
}

func TestNewElevenLabsOption_MissingKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"other": "value"})
	opt, err := NewElevenLabsOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "illegal vault config")
}

func TestNewElevenLabsOption_EmptyVault(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{})
	opt, err := NewElevenLabsOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
}

// --- Encoding Tests ---

func TestElevenLabsGetEncoding(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewElevenLabsOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.Equal(t, "pcm_16000", opt.GetEncoding())
}

// --- GetTextToSpeechConnectionString Tests ---

func TestGetTextToSpeechConnectionString_Default(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewElevenLabsOption(testutil.NewTestLogger(), cred, utils.Option{})
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "wss://api.elevenlabs.io/v1/text-to-speech/")
	assert.Contains(t, connStr, ELEVENLABS_VOICE_ID)
	assert.Contains(t, connStr, "output_format=pcm_16000")
	assert.Contains(t, connStr, "enable_ssml_parsing=true")
}

func TestGetTextToSpeechConnectionString_WithVoiceOverride(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "custom-voice-id",
	}
	opt, _ := NewElevenLabsOption(testutil.NewTestLogger(), cred, opts)
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "/custom-voice-id/multi-stream-input?")
	assert.NotContains(t, connStr, ELEVENLABS_VOICE_ID)
	assert.Contains(t, connStr, "output_format=pcm_16000")
}

func TestGetTextToSpeechConnectionString_WithLanguageAndModel(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.language": "fr",
		"speak.model":    "eleven_turbo_v2",
	}
	opt, _ := NewElevenLabsOption(testutil.NewTestLogger(), cred, opts)
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "language=fr")
	assert.Contains(t, connStr, "model_id=eleven_turbo_v2")
	assert.Contains(t, connStr, "output_format=pcm_16000")
}

func TestGetTextToSpeechConnectionString_AllOptions(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "my-voice",
		"speak.language": "es",
		"speak.model":    "eleven_multilingual_v2",
	}
	opt, _ := NewElevenLabsOption(testutil.NewTestLogger(), cred, opts)
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "/my-voice/multi-stream-input?")
	assert.Contains(t, connStr, "language=es")
	assert.Contains(t, connStr, "model_id=eleven_multilingual_v2")
	assert.Contains(t, connStr, "output_format=pcm_16000")
	assert.Contains(t, connStr, "enable_ssml_parsing=true")
}

func TestElevenLabsTextToSpeechWaitsForTextClosureAndFinalDrain(t *testing.T) {
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
	remote := waitElevenLabsServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &elevenlabsTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-eleven",
		logger:     testutil.NewTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-eleven",
		Text:      "first",
	}))
	assert.Equal(t, "first", waitElevenLabsRequest(t, requests)["text"])
	writeElevenLabsAudio(t, remote, []byte{1}, true)
	collector.WaitFor(t, time.Second, "first audio", func() bool {
		return len(collector.AudioPackets()) == 1
	})
	assert.Empty(t, collector.EndPackets())

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-eleven",
		Text:      "second",
	}))
	assert.Equal(t, "second", waitElevenLabsRequest(t, requests)["text"])
	writeElevenLabsAudio(t, remote, []byte{2}, true)
	collector.WaitFor(t, time.Second, "second audio", func() bool {
		return len(collector.AudioPackets()) == 2
	})
	assert.Empty(t, collector.EndPackets())

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-eleven",
	}))
	assert.Equal(t, " ", waitElevenLabsRequest(t, requests)["text"])
	writeElevenLabsAudio(t, remote, []byte{3}, true)
	collector.WaitForTTSEnd(t, time.Second)
	assert.Len(t, collector.AudioPackets(), 3)
	assert.Len(t, collector.EndPackets(), 1)
}

func TestElevenLabsProviderErrorDoesNotEmitEnd(t *testing.T) {
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
	remote := waitElevenLabsServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &elevenlabsTTS{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-eleven-error",
		logger:     testutil.NewTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, remote.WriteJSON(map[string]interface{}{"error": "provider rejected request"}))
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, collector.EndPackets())
}

func waitElevenLabsServerConnection(t *testing.T, ch <-chan *websocket.Conn) *websocket.Conn {
	t.Helper()
	select {
	case conn := <-ch:
		return conn
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket connection")
		return nil
	}
}

func waitElevenLabsRequest(t *testing.T, ch <-chan map[string]interface{}) map[string]interface{} {
	t.Helper()
	select {
	case request := <-ch:
		return request
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket request")
		return nil
	}
}

func writeElevenLabsAudio(t *testing.T, conn *websocket.Conn, audio []byte, final bool) {
	t.Helper()
	payload := map[string]interface{}{
		"audio":     base64.StdEncoding.EncodeToString(audio),
		"contextId": "ctx-eleven",
		"isFinal":   final,
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, data))
}
