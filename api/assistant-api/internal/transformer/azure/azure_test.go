package internal_transformer_azure

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/rapidaai/api/assistant-api/internal/observability"
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
	client := &azureTTSFakeClient{start: func(string, bool) (azureSynthesisStream, error) {
		return newAzureTTSStream(azureTTSRead{audio: []byte{1, 2}}, azureTTSRead{err: io.EOF}), nil
	}}
	tts, collector := newAzureTTSFixture(t, client)
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "ctx-azure", Text: "hello"}))
	assert.Empty(t, collector.EndPackets())
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "ctx-azure"}))
	assertAzureSynthesisEnd(t, collector.GetPackets(), "ctx-azure")
}

func TestAzureTextClosureWaitsForPendingSynthesis(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stream := newAzureTTSStream()
		stop := sync.OnceFunc(func() { close(stream.stopped) })
		client := &azureTTSFakeClient{
			start: func(string, bool) (azureSynthesisStream, error) { return stream, nil },
			stop:  func() error { stop(); return nil },
		}
		tts, collector := newAzureTTSFixture(t, client)
		textDone := make(chan error, 1)
		go func() {
			textDone <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "ctx-azure", Text: "hello"})
		}()
		<-stream.reading
		done := make(chan error, 1)
		go func() {
			done <- tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "ctx-azure"})
		}()
		synctest.Wait()
		require.Empty(t, done)
		require.Empty(t, collector.EndPackets())
		stream.chunks <- azureTTSRead{audio: []byte{1, 2}}
		stream.chunks <- azureTTSRead{audio: []byte{3, 4}, err: io.EOF}
		require.NoError(t, <-textDone)
		require.NoError(t, <-done)
		require.Len(t, collector.AudioPackets(), 2)
		require.Equal(t, []byte{1, 2}, collector.AudioPackets()[0].AudioChunk)
		require.Equal(t, []byte{3, 4}, collector.AudioPackets()[1].AudioChunk)
		assertAzureSynthesisEnd(t, collector.GetPackets(), "ctx-azure")
	})
}

func TestAzureSynthesisFailurePreventsEnd(t *testing.T) {
	client := &azureTTSFakeClient{start: func(string, bool) (azureSynthesisStream, error) {
		return nil, errors.New("synthesis failed")
	}}
	tts, collector := newAzureTTSFixture(t, client)
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "ctx-azure", Text: "hello"}))
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "ctx-azure"}))
	require.Empty(t, collector.EndPackets())
	require.Len(t, azureTTSErrors(collector), 1)
}

func assertAzureSynthesisEnd(t *testing.T, packets []internal_type.Packet, contextID string) {
	t.Helper()
	var terminal []internal_type.Packet
	for _, packet := range packets {
		switch packet := packet.(type) {
		case internal_type.TextToSpeechEndPacket:
			terminal = append(terminal, packet)
		case internal_type.ObservabilityEventRecordPacket:
			if packet.Record.Event == observability.TTSCompleted {
				terminal = append(terminal, packet)
			}
		}
	}
	packets = terminal
	assert.Len(t, packets, 2)
	endPacket, ok := packets[0].(internal_type.TextToSpeechEndPacket)
	assert.True(t, ok)
	assert.Equal(t, contextID, endPacket.ContextID)
	eventPacket, ok := packets[1].(internal_type.ObservabilityEventRecordPacket)
	assert.True(t, ok)
	assert.Equal(t, contextID, eventPacket.ContextID)
	assert.Equal(t, observability.TTSCompleted, eventPacket.Record.Event)
}
