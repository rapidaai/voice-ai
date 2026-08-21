// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package authenticators

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rapidaai/config"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

type projectAuthenticator struct {
	logger     commons.Logger
	cfg        *config.AppConfig
	authClient web_client.AuthClient
}

func NewProjectAuthenticator(cfg *config.AppConfig, logger commons.Logger, authClient web_client.AuthClient) types.ClaimAuthenticator[*types.ProjectScope] {
	return &projectAuthenticator{
		logger: logger, authClient: authClient, cfg: cfg,
	}
}

func (authenticator *projectAuthenticator) Claim(ctx context.Context, claimToken string) (*types.PlainClaimPrinciple[*types.ProjectScope], error) {
	start := time.Now()
	ath, err := authenticator.authClient.ScopeAuthorize(ctx, claimToken, "project")
	if err != nil {
		authenticator.logger.Debugf("error while claim %v", err)
		return nil, err
	}
	if ath.GetActorType() != string(types.ActorTypeProject) {
		return nil, fmt.Errorf("project authentication returned actor type %q", ath.GetActorType())
	}
	credentialID, err := strconv.ParseUint(ath.GetActorId(), 10, 64)
	if err != nil || credentialID == 0 {
		return nil, fmt.Errorf("project authentication returned invalid actor id")
	}
	authenticator.logger.Benchmark("Benchmarking: projectAuthenticator.Claim", time.Since(start))
	return &types.PlainClaimPrinciple[*types.ProjectScope]{
		Info: &types.ProjectScope{
			CredentialId:   &credentialID,
			OrganizationId: &ath.OrganizationId,
			ProjectId:      &ath.ProjectId,
			Status:         ath.GetStatus(),
			CurrentToken:   claimToken,
		},
	}, nil
}
