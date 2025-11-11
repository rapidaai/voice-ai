package internal_anthropic_callers

import (
	"context"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	integration_api "github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	Anthropic
}

func NewVerifyCredentialCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.Verifier {
	return &verifyCredentialCaller{
		Anthropic: anthropicAI(logger, credential),
	}
}

func (stc *verifyCredentialCaller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions) (*string, error) {
	return nil, nil

}
