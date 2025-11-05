package internal_replicate_callers

import (
	"context"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	Replicate
}

func NewVerifyCredentialCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.Verifier {
	return &verifyCredentialCaller{
		Replicate: replicate(logger, credential),
	}
}

func (stc *verifyCredentialCaller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions) (*string, error) {

	client, err := stc.GetClient()
	if err != nil {
		stc.logger.Errorf("validation unable to get client for cohere %v", err)
		return nil, err
	}

	// single minute timeout and cancellable by the client as context will get cancel
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	_, err = client.GetCurrentAccount(ctx)
	if err != nil {
		return nil, err
	}
	return utils.Ptr("valid"), nil

}
