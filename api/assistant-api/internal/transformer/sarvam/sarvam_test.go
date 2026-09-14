package internal_transformer_sarvam

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

func TestNewSarvamOption_ValidCredentials(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "test-api-key"})
	opt, err := NewSarvamOption(newTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "test-api-key", opt.GetKey())
}

func TestNewSarvamOption_MissingKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"other": "value"})
	opt, err := NewSarvamOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "missing 'key'")
}

func TestNewSarvamOption_EmptyVault(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{})
	opt, err := NewSarvamOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
}

// --- configureTextToSpeech Tests ---

func TestConfigureTextToSpeech_Defaults(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewSarvamOption(newTestLogger(), cred, utils.Option{})
	config := opt.configureTextToSpeech()

	assert.Equal(t, "config", config["type"])

	data, ok := config["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "en-IN", data["target_language_code"])
	assert.Equal(t, "anushka", data["speaker"])
	assert.Equal(t, 16000, data["speech_sample_rate"])
	assert.Equal(t, "linear16", data["output_audio_codec"])
}

func TestConfigureTextToSpeech_WithLanguageOverride(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.language": "hi-IN",
	}
	opt, _ := NewSarvamOption(newTestLogger(), cred, opts)
	config := opt.configureTextToSpeech()

	data := config["data"].(map[string]interface{})
	assert.Equal(t, "hi-IN", data["target_language_code"])
	assert.Equal(t, "anushka", data["speaker"]) // default speaker unchanged
	assert.Equal(t, 16000, data["speech_sample_rate"])
	assert.Equal(t, "linear16", data["output_audio_codec"])
}

func TestConfigureTextToSpeech_WithSpeakerOverride(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "meera",
	}
	opt, _ := NewSarvamOption(newTestLogger(), cred, opts)
	config := opt.configureTextToSpeech()

	data := config["data"].(map[string]interface{})
	assert.Equal(t, "meera", data["speaker"])
	assert.Equal(t, "en-IN", data["target_language_code"]) // default language unchanged
}

func TestConfigureTextToSpeech_WithAllOverrides(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.language": "ta-IN",
		"speak.voice.id": "meera",
	}
	opt, _ := NewSarvamOption(newTestLogger(), cred, opts)
	config := opt.configureTextToSpeech()

	data := config["data"].(map[string]interface{})
	assert.Equal(t, "ta-IN", data["target_language_code"])
	assert.Equal(t, "meera", data["speaker"])
	assert.Equal(t, 16000, data["speech_sample_rate"])
	assert.Equal(t, "linear16", data["output_audio_codec"])
}

// --- speechToTextMessage Tests ---

func TestSpeechToTextMessage_ValidInput(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewSarvamOption(newTestLogger(), cred, utils.Option{})

	input := []byte("hello audio data")
	result, err := opt.speechToTextMessage(input)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Parse the JSON output
	var payload map[string]interface{}
	err = json.Unmarshal(result, &payload)
	assert.NoError(t, err)

	audio, ok := payload["audio"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, audio["data"])
	assert.Equal(t, float64(16000), audio["sample_rate"])
	assert.Equal(t, "audio/wav", audio["encoding"])
	assert.Equal(t, "pcm_s16le", audio["input_audio_codec"])
}

func TestSpeechToTextMessage_EmptyInput(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewSarvamOption(newTestLogger(), cred, utils.Option{})

	result, err := opt.speechToTextMessage([]byte{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	var payload map[string]interface{}
	err = json.Unmarshal(result, &payload)
	assert.NoError(t, err)

	audio := payload["audio"].(map[string]interface{})
	assert.Equal(t, "", audio["data"]) // base64 of empty = ""
}

// --- URL Tests ---

func TestTextToSpeechUrl_Default(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewSarvamOption(newTestLogger(), cred, utils.Option{})
	url := opt.textToSpeechUrl()

	assert.Contains(t, url, TEXT_TO_SPEECH_URL)
	assert.Contains(t, url, "send_completion_event=true") // required for handleFlushComplete
	assert.NotContains(t, url, "model=")
}

func TestTextToSpeechUrl_WithModel(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.model": "bulbul:v2",
	}
	opt, _ := NewSarvamOption(newTestLogger(), cred, opts)
	url := opt.textToSpeechUrl()

	assert.Contains(t, url, TEXT_TO_SPEECH_URL)
	assert.Contains(t, url, "send_completion_event=true")
	assert.Contains(t, url, "model=bulbul")
}

func TestSarvamTextToSpeechWaitsForTextClosureAndFinalDrain(t *testing.T) {
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
	remote := waitSarvamTTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &sarvamTextToSpeech{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-sarvam",
		logger:     newTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-sarvam",
		Text:      "first",
	}))
	assert.Equal(t, "text", waitSarvamTTSRequest(t, requests)["type"])
	writeSarvamTTSAudio(t, remote, []byte{1})
	require.NoError(t, remote.WriteJSON(map[string]interface{}{
		"type": "event",
		"data": map[string]interface{}{"message": "complete"},
	}))
	collector.WaitFor(t, time.Second, "first audio", func() bool {
		return len(collector.AudioPackets()) == 1
	})
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, collector.EndPackets())

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-sarvam",
		Text:      "second",
	}))
	assert.Equal(t, "text", waitSarvamTTSRequest(t, requests)["type"])
	writeSarvamTTSAudio(t, remote, []byte{2})
	require.NoError(t, remote.WriteJSON(map[string]interface{}{
		"type": "event",
		"data": map[string]interface{}{"message": "complete"},
	}))
	collector.WaitFor(t, time.Second, "second audio", func() bool {
		return len(collector.AudioPackets()) == 2
	})
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, collector.EndPackets())

	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-sarvam",
	}))
	assert.Equal(t, "flush", waitSarvamTTSRequest(t, requests)["type"])
	writeSarvamTTSAudio(t, remote, []byte{3})
	require.NoError(t, remote.WriteJSON(map[string]interface{}{
		"type": "event",
		"data": map[string]interface{}{"message": "complete"},
	}))
	collector.WaitForTTSEnd(t, time.Second)
	assert.Len(t, collector.AudioPackets(), 3)
	assert.Len(t, collector.EndPackets(), 1)
}

func TestSarvamTextToSpeechProviderErrorDoesNotEmitEnd(t *testing.T) {
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
	remote := waitSarvamTTSServerConnection(t, serverConn)
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := testutil.NewPacketCollector()
	tts := &sarvamTextToSpeech{
		ctx:        ctx,
		ctxCancel:  cancel,
		connection: conn,
		contextId:  "ctx-sarvam-error",
		logger:     newTestLogger(),
		onPacket:   collector.OnPacket,
	}
	go tts.readLoop(conn)

	require.NoError(t, remote.WriteJSON(map[string]interface{}{
		"type": "error",
		"data": map[string]interface{}{"message": "provider rejected request"},
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

func waitSarvamTTSServerConnection(t *testing.T, ch <-chan *websocket.Conn) *websocket.Conn {
	t.Helper()
	select {
	case conn := <-ch:
		return conn
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket connection")
		return nil
	}
}

func waitSarvamTTSRequest(t *testing.T, ch <-chan map[string]interface{}) map[string]interface{} {
	t.Helper()
	select {
	case request := <-ch:
		return request
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket request")
		return nil
	}
}

func writeSarvamTTSAudio(t *testing.T, conn *websocket.Conn, audio []byte) {
	t.Helper()
	require.NoError(t, conn.WriteJSON(map[string]interface{}{
		"type": "audio",
		"data": map[string]interface{}{
			"audio": base64.StdEncoding.EncodeToString(audio),
		},
	}))
}

func TestSpeechToTextUrl_Default(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewSarvamOption(newTestLogger(), cred, utils.Option{})
	url := opt.speechToTextUrl()

	assert.Contains(t, url, SPEECH_TO_TEXT_URL)
	assert.Contains(t, url, "sample_rate=16000")
	assert.Contains(t, url, "input_audio_codec=pcm_s16le")
	assert.NotContains(t, url, "language-code=")
	assert.NotContains(t, url, "model=")
}

func TestSpeechToTextUrl_WithLanguageAndModel(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"listen.language": "hi-IN",
		"listen.model":    "saaras:v2",
	}
	opt, _ := NewSarvamOption(newTestLogger(), cred, opts)
	url := opt.speechToTextUrl()

	assert.Contains(t, url, SPEECH_TO_TEXT_URL)
	assert.Contains(t, url, "sample_rate=16000")
	assert.Contains(t, url, "input_audio_codec=pcm_s16le")
	assert.Contains(t, url, "language-code=hi-IN")
	assert.Contains(t, url, "model=saaras")
}
