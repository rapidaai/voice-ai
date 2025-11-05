package internal_huggingface_callers

import (
	"context"
	"net/http"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	Huggingface
}

func NewVerifyCredentialCaller(logger commons.Logger,
	credential *integration_api.Credential) internal_callers.Verifier {
	return &verifyCredentialCaller{
		Huggingface: huggingface(logger, AUTH_URL, credential),
	}
}

func (stc *verifyCredentialCaller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions) (*string, error) {
	//
	headers := map[string]string{}
	_, err := stc.Call(ctx, "api/whoami-v2", "GET", headers, nil)
	if err != nil {
		stc.logger.Errorf("error occured while calling verification api for Huggingface %v", err)
		if ve, ok := err.(HuggingfaceError); ok {
			if ve.StatusCode == http.StatusForbidden || ve.StatusCode == http.StatusUnauthorized {
				return nil, err
			}
			return utils.Ptr("valid"), nil
		}
		return nil, err
	}
	return nil, err
}
