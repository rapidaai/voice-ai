package internal_cohere_callers

import (
	"context"
	"errors"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	Cohere
}

func NewVerifyCredentialCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.Verifier {
	return &verifyCredentialCaller{
		Cohere: NewCohere(logger, credential),
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
	//
	response, err := client.CheckApiKey(
		ctx,
	)
	if err != nil {
		return nil, err
	}
	if response.Valid {
		return utils.Ptr("valid"), nil
	}
	return nil, errors.New("given credential is not verified by cohere")

}
