// Rapida -- Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_minimax_callers

import (
	"context"
	"net/http"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	MiniMax
}

func NewVerifyCredentialCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.Verifier {
	return &verifyCredentialCaller{
		MiniMax: minimax(logger, credential),
	}
}

func (vc *verifyCredentialCaller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions) (*string, error) {
	// MiniMax does not have a /v1/models endpoint.
	// Verify credentials by making a minimal chat completion request.
	payload := map[string]interface{}{
		"model": "MiniMax-M2.7",
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
		"max_tokens":  1,
		"temperature": 0.01,
	}
	_, err := vc.CallJSON(ctx, "chat/completions", "POST", map[string]string{}, payload)
	if err != nil {
		vc.logger.Debugf("minimax credential verification with error %v", err)
		// Check if the error indicates auth failure specifically
		if resp, callErr := vc.Call(ctx, "chat/completions", "POST", map[string]string{}, payload); callErr == nil {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
				return utils.Ptr("valid"), nil
			}
		}
		return nil, err
	}
	return utils.Ptr("valid"), nil
}
