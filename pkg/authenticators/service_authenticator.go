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
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
)

type serviceAuthenticator struct {
	logger commons.Logger
	cfg    *config.AppConfig
}

func NewServiceAuthenticator(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) types.ClaimAuthenticator[*types.ServiceScope] {
	return &serviceAuthenticator{
		logger: logger, cfg: cfg,
	}
}

func (authenticator *serviceAuthenticator) Claim(ctx context.Context, claimToken string) (*types.PlainClaimPrinciple[*types.ServiceScope], error) {
	start := time.Now()
	serviceScope, err := types.ExtractServiceScope(claimToken, authenticator.cfg.Secret)
	if err != nil {
		authenticator.logger.Errorf("authentication error for user %v", err)
		return nil, err
	}
	authenticator.logger.Debugf("Benchmarking: serviceAuthenticator.Claim time taken %v", time.Since(start))
	return &types.PlainClaimPrinciple[*types.ServiceScope]{
		Info: serviceScope,
	}, nil
}
