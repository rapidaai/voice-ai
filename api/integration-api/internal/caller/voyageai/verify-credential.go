package internal_voyageai_callers

import (
	"context"
	"net/http"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	Voyageai
}

func NewVerifyCredentialCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.Verifier {
	return &verifyCredentialCaller{
		Voyageai: voyageai(logger, credential),
	}
}

func (stc *verifyCredentialCaller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions) (*string, error) {
	request := map[string]interface{}{
		"input": []string{"test"},
		"model": "test",
	}
	headers := map[string]string{}
	_, err := stc.Call(ctx, "embeddings", "POST", headers, request)
	if err != nil {
		stc.logger.Errorf("error occured while calling verification api for voyageai %v", err)
		if ve, ok := err.(VoyageaiError); ok {
			if ve.StatusCode != http.StatusUnauthorized {
				return utils.Ptr("valid"), nil
			}
		}
		return nil, err
	}
	return nil, err
}
