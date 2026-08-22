// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package authenticators

import (
	"context"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

type serviceAuthenticator struct {
	logger commons.Logger
	secret string
}

func NewServiceAuthenticator(cfg *config.AppConfig, logger commons.Logger) types.ClaimAuthenticator[*types.ServiceScope] {
	return &serviceAuthenticator{logger: logger, secret: cfg.Secret}
}

func (authenticator *serviceAuthenticator) Claim(_ context.Context, claimToken string) (*types.PlainClaimPrinciple[*types.ServiceScope], error) {
	serviceScope, err := types.ExtractServiceScope(claimToken, authenticator.secret)
	if err != nil {
		authenticator.logger.Errorf("service authentication failed: %v", err)
		return nil, err
	}
	return &types.PlainClaimPrinciple[*types.ServiceScope]{Info: serviceScope}, nil
}
