// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package authenticators

import (
	"context"
	"fmt"
	"math"
	"strconv"
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
	ath, err := authenticator.authClient.ScopeAuthorize(ctx, claimToken, "organization")
	if err != nil {
		return nil, err
	}
	if ath.GetActorType() != string(types.ActorTypeOrganization) {
		return nil, fmt.Errorf("organization authentication returned actor type %q", ath.GetActorType())
	}
	credentialID, err := strconv.ParseUint(ath.GetActorId(), 10, 64)
	if err != nil || credentialID == 0 || credentialID > math.MaxInt64 {
		return nil, fmt.Errorf("organization authentication returned invalid actor id")
	}

	authenticator.logger.Debugf("Benchmarking: organizationAuthenticator.Claim time taken %v", time.Since(start))
	return &types.PlainClaimPrinciple[*types.OrganizationScope]{
		Info: &types.OrganizationScope{
			CredentialId:   &credentialID,
			OrganizationId: &ath.OrganizationId,
			Status:         ath.GetStatus(),
			CurrentToken:   claimToken,
		},
	}, nil
}
