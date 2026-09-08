package internal_transformer_cartesia

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
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

func TestNewCartesiaOption_ValidCredentials(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "test-api-key"})
	opt, err := NewCartesiaOption(newTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "test-api-key", opt.key)
}

func TestNewCartesiaOption_MissingKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"other": "value"})
	opt, err := NewCartesiaOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "unable to get config parameters")
}

func TestNewCartesiaOption_EmptyVault(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{})
	opt, err := NewCartesiaOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
}

func TestCartesiaGetEncoding(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewCartesiaOption(newTestLogger(), cred, utils.Option{})
	assert.Equal(t, "pcm_s16le", opt.GetEncoding())
}

func TestGetTextToSpeechInput_Defaults(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewCartesiaOption(newTestLogger(), cred, utils.Option{})
	input := opt.GetTextToSpeechInput("hello world", map[string]interface{}{})
	assert.Equal(t, "hello world", input.Transcript)
	assert.Equal(t, "sonic-2-2025-03-07", input.ModelID)
	assert.Equal(t, "id", input.Voice.Mode)
	assert.Equal(t, "c2ac25f9-ecc4-4f56-9095-651354df60c0", input.Voice.ID)
	assert.Equal(t, "raw", input.OutputFormat.Container)
	assert.Equal(t, "pcm_s16le", input.OutputFormat.Encoding)
	assert.Equal(t, 16000, input.OutputFormat.SampleRate)
	assert.False(t, input.AddTimestamps)
}

func TestGetTextToSpeechInput_WithOverrides(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "custom-voice-id",
		"speak.model":    "sonic-mini",
		"speak.language": "fr",
	}
	opt, _ := NewCartesiaOption(newTestLogger(), cred, opts)
	input := opt.GetTextToSpeechInput("bonjour", map[string]interface{}{})
	assert.Equal(t, "bonjour", input.Transcript)
	assert.Equal(t, "sonic-mini", input.ModelID)
	assert.Equal(t, "custom-voice-id", input.Voice.ID)
	assert.Equal(t, "fr", input.Language)
	assert.Equal(t, "pcm_s16le", input.OutputFormat.Encoding)
	assert.Equal(t, 16000, input.OutputFormat.SampleRate)
}

func TestGetTextToSpeechInput_WithContinueAndContextID(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewCartesiaOption(newTestLogger(), cred, utils.Option{})
	overrides := map[string]interface{}{
		"continue":   true,
		"context_id": "ctx-123",
	}
	input := opt.GetTextToSpeechInput("hello", overrides)
	assert.True(t, input.Continue)
	assert.Equal(t, "ctx-123", input.ContextID)
}

func TestGetTextToSpeechInput_WithExperimentalControls(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.__experimental_controls.speed":   "fast",
		"speak.__experimental_controls.emotion": "happy<|||>excited",
	}
	opt, _ := NewCartesiaOption(newTestLogger(), cred, opts)
	input := opt.GetTextToSpeechInput("test", map[string]interface{}{})
	assert.Equal(t, "fast", input.ExperimentalControls.Speed)
	assert.Equal(t, []string{"happy", "excited"}, input.ExperimentalControls.Emotion)
}

func TestGetSpeechToTextConnectionString_Default(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "my-key"})
	opt, _ := NewCartesiaOption(newTestLogger(), cred, utils.Option{})
	connStr := opt.GetSpeechToTextConnectionString()
	assert.Contains(t, connStr, "wss://api.cartesia.ai/stt/websocket?")
	assert.NotContains(t, connStr, "api_key=")
	assert.NotContains(t, connStr, "cartesia_version=")
	assert.Contains(t, connStr, "encoding=pcm_s16le")
	assert.Contains(t, connStr, "model=ink-whisper")
	assert.Contains(t, connStr, "sample_rate=16000")
}

func TestGetSpeechToTextConnectionString_WithLanguageAndModel(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "my-key"})
	opts := utils.Option{
		"listen.language": "fr",
		"listen.model":    "nova-2",
	}
	opt, _ := NewCartesiaOption(newTestLogger(), cred, opts)
	connStr := opt.GetSpeechToTextConnectionString()
	assert.Contains(t, connStr, "language=fr")
	assert.Contains(t, connStr, "model=nova-2")
	assert.Contains(t, connStr, "encoding=pcm_s16le")
	assert.Contains(t, connStr, "sample_rate=16000")
}

func TestGetSpeechToTextHeader(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "my-key"})
	opt, _ := NewCartesiaOption(newTestLogger(), cred, utils.Option{})

	header := opt.GetSpeechToTextHeader()

	assert.Equal(t, "my-key", header.Get("X-API-Key"))
	assert.Equal(t, CARTESIA_STT_API_VERSION, header.Get("Cartesia-Version"))
}

func TestCartesiaSpeechToTextTransform_ContextAndFinalize(t *testing.T) {
	upgrader := websocket.Upgrader{}
	messages := make(chan struct {
		messageType int
		payload     string
	}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()
		for i := 0; i < 2; i++ {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			messages <- struct {
				messageType int
				payload     string
			}{messageType: messageType, payload: string(payload)}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	stt := &cartesiaSpeechToText{
		connection: conn,
		logger:     newTestLogger(),
		onPacket:   func(pkt ...internal_type.Packet) error { return nil },
	}

	require.NoError(t, stt.Transform(context.Background(), internal_type.SpeechToTextStartPacket{
		ContextID: "ctx-start",
	}))
	require.NoError(t, stt.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-audio",
		Audio:     []byte{1, 2, 3, 4},
	}))
	require.NoError(t, stt.Transform(context.Background(), internal_type.SpeechToTextEndPacket{
		ContextID: "ctx-end",
	}))

	readMessage := func(label string) struct {
		messageType int
		payload     string
	} {
		select {
		case message := <-messages:
			return message
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s message", label)
			return struct {
				messageType int
				payload     string
			}{}
		}
	}

	audioMessage := readMessage("audio")
	assert.Equal(t, websocket.BinaryMessage, audioMessage.messageType)
	assert.Equal(t, string([]byte{1, 2, 3, 4}), audioMessage.payload)

	finalizeMessage := readMessage("finalize")
	assert.Equal(t, websocket.TextMessage, finalizeMessage.messageType)
	assert.Equal(t, "finalize", finalizeMessage.payload)
	assert.Equal(t, "ctx-end", stt.contextId)
}

func TestGetTextToSpeechConnectionString(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "my-key"})
	opt, _ := NewCartesiaOption(newTestLogger(), cred, utils.Option{})
	connStr := opt.GetTextToSpeechConnectionString()
	assert.Contains(t, connStr, "wss://api.cartesia.ai/tts/websocket?")
	assert.Contains(t, connStr, "api_key=my-key")
	assert.Contains(t, connStr, "cartesia_version="+CARTESIA_API_VERSION)
}
