package internal_transformer_smallest

import (
	"net/http"
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
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

func TestNewSmallestOption_ValidCredentials(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "test-api-key"})
	opt, err := NewSmallestOption(newTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "test-api-key", opt.key)
	assert.Equal(t, "test-api-key", opt.GetKey())
}

func TestNewSmallestOption_MissingKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"other": "value"})
	opt, err := NewSmallestOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "missing 'key'")
}

func TestNewSmallestOption_EmptyVault(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{})
	opt, err := NewSmallestOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
}

func TestNewSmallestOption_NonStringKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": 12345})
	opt, err := NewSmallestOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "must be a string")
}

func TestSmallestGetEncoding(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewSmallestOption(newTestLogger(), cred, utils.Option{})
	assert.Equal(t, "linear16", opt.GetEncoding())
}

func TestGetTextToSpeechInput_Defaults(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewSmallestOption(newTestLogger(), cred, utils.Option{})
	input := opt.GetTextToSpeechInput("hello world", map[string]interface{}{})
	assert.Equal(t, "hello world", input.Text)
	assert.Equal(t, DefaultTextToSpeechModel, input.Model)
	assert.Equal(t, DefaultVoiceID, input.VoiceID)
	assert.Equal(t, DefaultSampleRate, input.SampleRate)
	assert.False(t, input.Continue)
	assert.False(t, input.Flush)
}

func TestGetTextToSpeechInput_WithOverrides(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "meher",
		"speak.model":    "lightning_v3.1_pro",
		"speak.language": "hi",
	}
	opt, _ := NewSmallestOption(newTestLogger(), cred, opts)
	input := opt.GetTextToSpeechInput("namaste", map[string]interface{}{})
	assert.Equal(t, "namaste", input.Text)
	assert.Equal(t, "lightning_v3.1_pro", input.Model)
	assert.Equal(t, "meher", input.VoiceID)
	assert.Equal(t, "hi", input.Language)
	assert.Equal(t, DefaultSampleRate, input.SampleRate)
}

func TestGetTextToSpeechInput_WithSpeed(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{"speak.speed": "1.25"}
	opt, _ := NewSmallestOption(newTestLogger(), cred, opts)
	input := opt.GetTextToSpeechInput("hello", map[string]interface{}{})
	assert.Equal(t, 1.25, input.Speed)
}

func TestGetTextToSpeechInput_WithContinueFlushAndContextID(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewSmallestOption(newTestLogger(), cred, utils.Option{})

	streaming := opt.GetTextToSpeechInput("hello", map[string]interface{}{
		"continue":   true,
		"context_id": "ctx-123",
	})
	assert.True(t, streaming.Continue)
	assert.False(t, streaming.Flush)
	assert.Equal(t, "ctx-123", streaming.ContextID)

	done := opt.GetTextToSpeechInput("", map[string]interface{}{
		"continue":   false,
		"flush":      true,
		"context_id": "ctx-123",
	})
	assert.False(t, done.Continue)
	assert.True(t, done.Flush)
	assert.Equal(t, "ctx-123", done.ContextID)
}

func TestGetSpeechToTextConnectionString_Default(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "my-key"})
	opt, _ := NewSmallestOption(newTestLogger(), cred, utils.Option{})
	connStr := opt.GetSpeechToTextConnectionString()
	assert.Contains(t, connStr, "wss://api.smallest.ai/waves/v1/pulse/get_text?")
	assert.Contains(t, connStr, "encoding=linear16")
	assert.Contains(t, connStr, "sample_rate=16000")
	assert.NotContains(t, connStr, "api_key")
	assert.NotContains(t, connStr, "language=")
}

func TestGetSpeechToTextConnectionString_WithLanguage(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "my-key"})
	opts := utils.Option{"listen.language": "hi"}
	opt, _ := NewSmallestOption(newTestLogger(), cred, opts)
	connStr := opt.GetSpeechToTextConnectionString()
	assert.Contains(t, connStr, "language=hi")
	assert.Contains(t, connStr, "encoding=linear16")
	assert.Contains(t, connStr, "sample_rate=16000")
}

func TestGetSpeechToTextConnectionString_Default_OmitsFeatureFlags(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "my-key"})
	opt, _ := NewSmallestOption(newTestLogger(), cred, utils.Option{})
	connStr := opt.GetSpeechToTextConnectionString()
	for _, absent := range []string{"word_timestamps", "sentence_timestamps", "diarize", "redact_pii", "redact_pci", "numerals", "format"} {
		assert.NotContains(t, connStr, absent+"=", "should not set %s unless explicitly requested", absent)
	}
}

func TestGetSpeechToTextConnectionString_WithFormat(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "my-key"})
	opts := utils.Option{"listen.smart_format": false}
	opt, _ := NewSmallestOption(newTestLogger(), cred, opts)
	connStr := opt.GetSpeechToTextConnectionString()
	assert.Contains(t, connStr, "format=false")
}

func TestGetSpeechToTextConnectionString_WithFeatureFlags(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "my-key"})
	opts := utils.Option{
		"listen.word_timestamps":     true,
		"listen.sentence_timestamps": true,
		"listen.diarize":             true,
		"listen.redact_pii":          true,
		"listen.redact_pci":          false,
		"listen.numerals":            "auto",
	}
	opt, _ := NewSmallestOption(newTestLogger(), cred, opts)
	connStr := opt.GetSpeechToTextConnectionString()
	assert.Contains(t, connStr, "word_timestamps=true")
	assert.Contains(t, connStr, "sentence_timestamps=true")
	assert.Contains(t, connStr, "diarize=true")
	assert.Contains(t, connStr, "redact_pii=true")
	assert.Contains(t, connStr, "redact_pci=false")
	assert.Contains(t, connStr, "numerals=auto")
}

func TestGetTextToSpeechConnectionString(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "my-key"})
	opt, _ := NewSmallestOption(newTestLogger(), cred, utils.Option{})
	assert.Equal(t, "wss://api.smallest.ai/waves/v1/tts/live", opt.GetTextToSpeechConnectionString())
}

func TestSetSourceHeaders(t *testing.T) {
	header := http.Header{}
	setSourceHeaders(header)
	assert.Equal(t, "rapida", header.Get("X-Source"))
}
