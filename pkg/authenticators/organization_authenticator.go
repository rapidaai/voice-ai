// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package authenticators

import (
	"context"
	"time"

	"github.com/rapidaai/config"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

type organizationAuthenticator struct {
	logger commons.Logger

	cfg        *config.AppConfig
	authClient web_client.AuthClient
}

func NewOrganizationAuthenticator(cfg *config.AppConfig, logger commons.Logger, authClient web_client.AuthClient) types.ClaimAuthenticator[*types.OrganizationScope] {
	return &organizationAuthenticator{
		logger: logger, authClient: authClient, cfg: cfg,
	}
}

func (authenticator *organizationAuthenticator) Claim(ctx context.Context, claimToken string) (*types.PlainClaimPrinciple[*types.OrganizationScope], error) {
	start := time.Now()
	ath, err := authenticator.authClient.ScopeAuthorize(ctx, claimToken, "organizaiton")
	if err != nil {
		return nil, err
	}

	authenticator.logger.Debugf("Benchmarking: organizationAuthenticator.Claim time taken %v", time.Since(start))
	return &types.PlainClaimPrinciple[*types.OrganizationScope]{
		Info: &types.OrganizationScope{
			OrganizationId: &ath.OrganizationId,
			Status:         ath.GetStatus(),
			CurrentToken:   claimToken,
		},
	}, nil

}
