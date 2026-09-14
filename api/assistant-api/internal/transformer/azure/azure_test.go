package internal_transformer_azure

import (
	"testing"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
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

// --- Constructor Tests ---

func TestNewAzureOption_ValidCredentials(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{
		"subscription_key": "test-sub-key",
		"endpoint":         "https://test.cognitiveservices.azure.com",
	})
	opt, err := NewAzureOption(newTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "test-sub-key", opt.subscriptionKey)
	assert.Equal(t, "https://test.cognitiveservices.azure.com", opt.endpoint)
}

func TestNewAzureOption_MissingSubscriptionKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{
		"endpoint": "https://test.cognitiveservices.azure.com",
	})
	opt, err := NewAzureOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "subscription_key")
}

func TestNewAzureOption_MissingEndpoint(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{
		"subscription_key": "test-sub-key",
	})
	opt, err := NewAzureOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "endpoint")
}

func TestNewAzureOption_EmptyVault(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{})
	opt, err := NewAzureOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
}

// --- Output Format Tests ---

func TestGetSpeechSynthesisOutputFormat(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{
		"subscription_key": "k",
		"endpoint":         "https://e.azure.com",
	})
	opt, _ := NewAzureOption(newTestLogger(), cred, utils.Option{})
	assert.Equal(t, common.Raw16Khz16BitMonoPcm, opt.GetSpeechSynthesisOutputFormat())
}

// --- Audio Stream Format Tests ---

func TestGetAudioStreamFormat(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{
		"subscription_key": "k",
		"endpoint":         "https://e.azure.com",
	})
	opt, _ := NewAzureOption(newTestLogger(), cred, utils.Option{})
	format := opt.GetAudioStreamFormat()
	assert.NotNil(t, format)
}

func TestAzureSynthesisCompletionWaitsForTextClosure(t *testing.T) {
	var packets []internal_type.Packet
	tts := &azureTextToSpeech{
		contextId: "ctx-azure",
		onPacket: func(pkts ...internal_type.Packet) error {
			packets = append(packets, pkts...)
			return nil
		},
	}

	tts.beginSynthesis("ctx-azure")
	tts.finishSynthesis("ctx-azure", false)

	assert.Empty(t, packets)

	tts.closeText("ctx-azure")

	assertAzureSynthesisEnd(t, packets, "ctx-azure")
}

func TestAzureTextClosureWaitsForPendingSynthesis(t *testing.T) {
	var packets []internal_type.Packet
	tts := &azureTextToSpeech{
		contextId: "ctx-azure",
		onPacket: func(pkts ...internal_type.Packet) error {
			packets = append(packets, pkts...)
			return nil
		},
	}

	tts.beginSynthesis("ctx-azure")
	tts.closeText("ctx-azure")

	assert.Empty(t, packets)

	tts.finishSynthesis("ctx-azure", false)

	assertAzureSynthesisEnd(t, packets, "ctx-azure")
}

func TestAzureSynthesisFailurePreventsEnd(t *testing.T) {
	var packets []internal_type.Packet
	tts := &azureTextToSpeech{
		contextId: "ctx-azure",
		onPacket: func(pkts ...internal_type.Packet) error {
			packets = append(packets, pkts...)
			return nil
		},
	}

	tts.beginSynthesis("ctx-azure")
	tts.finishSynthesis("ctx-azure", true)
	tts.closeText("ctx-azure")

	assert.Empty(t, packets)
}

func assertAzureSynthesisEnd(t *testing.T, packets []internal_type.Packet, contextID string) {
	t.Helper()
	assert.Len(t, packets, 2)
	endPacket, ok := packets[0].(internal_type.TextToSpeechEndPacket)
	assert.True(t, ok)
	assert.Equal(t, contextID, endPacket.ContextID)
	eventPacket, ok := packets[1].(internal_type.ObservabilityEventRecordPacket)
	assert.True(t, ok)
	assert.Equal(t, contextID, eventPacket.ContextID)
	assert.Equal(t, observability.TTSCompleted, eventPacket.Record.Event)
}
