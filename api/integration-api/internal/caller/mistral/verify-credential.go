package internal_mistral_callers

import (
	"context"
	"net/http"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	Mistral
}

func NewVerifyCredentialCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.Verifier {
	return &verifyCredentialCaller{
		Mistral: mistral(logger, credential),
	}
}

func (stc *verifyCredentialCaller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions) (*string, error) {

	headers := map[string]string{}
	_, err := stc.Call(ctx, "v1/models", "GET", headers, nil)
	if err != nil {
		stc.logger.Debugf("mistral with error %v", err)
		if httpError, ok := err.(*MistralError); ok {
			if httpError.StatusCode != http.StatusUnauthorized {
				return utils.Ptr("valid"), nil
			}
		}
		return nil, err
	}
	return nil, err

}
