package internal_google_callers

import (
	"context"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	Google
}

func NewVerifyCredentialCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.Verifier {
	return &verifyCredentialCaller{
		Google: google(logger, credential),
	}
}

func (llc *verifyCredentialCaller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions) (*string, error) {
	return utils.Ptr("valid"), nil

}
