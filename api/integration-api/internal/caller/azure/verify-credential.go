package internal_azure_callers

import (
	"context"
	"time"

	"github.com/openai/openai-go"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	integration_api "github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	AzureAi
}

func NewVerifyCredentialCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.Verifier {
	return &verifyCredentialCaller{
		AzureAi: azure(logger, credential),
	}
}

func (stc *verifyCredentialCaller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions) (*string, error) {
	client, err := stc.GetClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	_, err = client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Test"),
		},
		Model: openai.ChatModelGPT4o,
	})
	if err != nil {
		return nil, err
	}
	return nil, err

}
