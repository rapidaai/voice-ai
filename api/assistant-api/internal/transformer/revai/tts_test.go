package internal_transformer_revai

import (
	"context"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRevaiTextToSpeechUnsupported(t *testing.T) {
	value, err := structpb.NewStruct(map[string]interface{}{"key": "test-api-key"})
	require.NoError(t, err)
	for _, scenario := range []struct {
		name       string
		credential *protos.VaultCredential
	}{
		{name: "missing credentials"},
		{name: "configured credentials", credential: &protos.VaultCredential{Value: value}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			transformer, err := NewRevaiTextToSpeech(context.Background(), nil, scenario.credential,
				func(...internal_type.Packet) error {
					t.Fatal("unsupported provider must not emit packets")
					return nil
				}, utils.Option{})
			require.Nil(t, transformer)
			require.EqualError(t, err, "revai-tts: text-to-speech is not supported")
		})
	}
}
