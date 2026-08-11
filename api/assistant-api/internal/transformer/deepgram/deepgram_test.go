package internal_transformer_deepgram

import (
	"testing"

	deepgram_internal "github.com/rapidaai/api/assistant-api/internal/transformer/deepgram/internal"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func newVaultCredential(m map[string]interface{}) *protos.VaultCredential {
	val, _ := structpb.NewStruct(m)
	return &protos.VaultCredential{Value: val}
}

// --- Constructor Tests ---

func TestNewDeepgramOption_ValidCredentials(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "test-api-key"})
	opt, err := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "test-api-key", opt.GetKey())
	assert.Equal(t, "api.deepgram.com", opt.GetEndpoint())
}

func TestNewDeepgramOption_MissingKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"other": "value"})
	opt, err := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "illegal vault config")
}

func TestNewDeepgramOption_EmptyVault(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{})
	opt, err := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
}

func TestNewDeepgramOption_WithEndpoint(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{
		"key":      "test-api-key",
		"endpoint": "api.eu.deepgram.com",
	})
	opt, err := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "api.eu.deepgram.com", opt.GetEndpoint())
	assert.Equal(t, "api.eu.deepgram.com", opt.ClientOptions().Host)
}

func TestNewDeepgramOption_UsesEndpointAsProvided(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{
		"key":      "test-api-key",
		"endpoint": "wss://api.au.deepgram.com/v1/speak",
	})
	opt, err := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "wss://api.au.deepgram.com/v1/speak", opt.GetEndpoint())
}

// --- Encoding Tests ---

func TestDeepgramGetEncoding(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.Equal(t, "linear16", opt.GetEncoding())
}

// --- SpeechToTextOptions Tests ---

func TestSpeechToTextOptions_Defaults(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, utils.Option{})
	sttOpts := opt.SpeechToTextOptions()

	assert.Equal(t, "nova", sttOpts.Model)
	assert.Equal(t, "en-US", sttOpts.Language)
	assert.Equal(t, 1, sttOpts.Channels)
	assert.True(t, sttOpts.SmartFormat)
	assert.True(t, sttOpts.InterimResults)
	assert.True(t, sttOpts.FillerWords)
	assert.False(t, sttOpts.VadEvents)
	assert.Equal(t, "5", sttOpts.Endpointing)
	assert.True(t, sttOpts.Punctuate)
	assert.True(t, sttOpts.NoDelay)
	assert.Equal(t, "linear16", sttOpts.Encoding)
	assert.Equal(t, 16000, sttOpts.SampleRate)
	assert.False(t, sttOpts.Diarize)
	assert.False(t, sttOpts.Multichannel)
	assert.Empty(t, sttOpts.Tag)
}

func TestSpeechToTextOptions_WithOverrides(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"listen.language":     "fr-FR",
		"listen.smart_format": false,
		"listen.filler_words": false,
		"listen.vad_events":   true,
		"listen.endpointing":  "10",
		"listen.punctuate":    false,
		"listen.diarize":      true,
		"listen.multichannel": true,
		"listen.model":        "nova-2",
	}
	opt, _ := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, opts)
	sttOpts := opt.SpeechToTextOptions()

	assert.Equal(t, "fr-FR", sttOpts.Language)
	assert.False(t, sttOpts.SmartFormat)
	assert.False(t, sttOpts.FillerWords)
	assert.True(t, sttOpts.VadEvents)
	assert.Equal(t, "10", sttOpts.Endpointing)
	assert.False(t, sttOpts.Punctuate)
	assert.True(t, sttOpts.Diarize)
	assert.True(t, sttOpts.Multichannel)
	assert.Equal(t, "nova-2", sttOpts.Model)
	// Encoding and sample rate remain hardcoded
	assert.Equal(t, "linear16", sttOpts.Encoding)
	assert.Equal(t, 16000, sttOpts.SampleRate)
}

func TestSpeechToTextOptions_WithTags(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := deepgram_internal.NewDeepgramOption(
		testutil.NewTestLogger(),
		cred,
		utils.Option{},
		"agent:123",
		"conversation:456",
	)
	sttOpts := opt.SpeechToTextOptions()

	assert.Equal(t, []string{"agent:123", "conversation:456"}, sttOpts.Tag)
}

func TestSpeechToTextOptions_KeywordsNova2(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"listen.model":   "nova-2",
		"listen.keyword": []interface{}{"hello", "world"},
	}
	opt, _ := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, opts)
	sttOpts := opt.SpeechToTextOptions()

	assert.Equal(t, []string{"hello", "world"}, sttOpts.Keywords)
	assert.Empty(t, sttOpts.Keyterm)
}

func TestSpeechToTextOptions_KeywordsNova3(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"listen.model":   "nova-3",
		"listen.keyword": []interface{}{"alpha", "beta"},
	}
	opt, _ := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, opts)
	sttOpts := opt.SpeechToTextOptions()

	assert.Equal(t, []string{"alpha", "beta"}, sttOpts.Keyterm)
	assert.Empty(t, sttOpts.Keywords)
}

func TestSpeechToTextOptions_KeywordsAsString(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"listen.model":   "nova-2",
		"listen.keyword": "[hello world]",
	}
	opt, _ := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, opts)
	sttOpts := opt.SpeechToTextOptions()

	assert.Equal(t, []string{"hello", "world"}, sttOpts.Keywords)
}

// --- TextToSpeech Connection String Tests ---

func TestGetTextToSpeechConnectionString_Default(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, utils.Option{})
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "wss://api.deepgram.com/v1/speak?")
	assert.Contains(t, connStr, "encoding=linear16")
	assert.Contains(t, connStr, "sample_rate=16000")
	assert.NotContains(t, connStr, "model=")
}

func TestGetTextToSpeechConnectionString_WithVoice(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "aura-asteria-en",
	}
	opt, _ := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, opts)
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "wss://api.deepgram.com/v1/speak?")
	assert.Contains(t, connStr, "encoding=linear16")
	assert.Contains(t, connStr, "sample_rate=16000")
	assert.Contains(t, connStr, "model=aura-asteria-en")
}

func TestGetTextToSpeechConnectionString_WithEndpoint(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{
		"key":      "k",
		"endpoint": "api.eu.deepgram.com",
	})
	opt, _ := deepgram_internal.NewDeepgramOption(testutil.NewTestLogger(), cred, utils.Option{})
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "wss://api.eu.deepgram.com/v1/speak?")
	assert.Contains(t, connStr, "encoding=linear16")
	assert.Contains(t, connStr, "sample_rate=16000")
}
