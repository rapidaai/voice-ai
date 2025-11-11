package authenticators

import (
	"context"
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
		authenticator.logger.Errorf("error while claim %v", err)
		return nil, err
	}
	authenticator.logger.Benchmark("Benchmarking: projectAuthenticator.Claim", time.Since(start))
	return &types.PlainClaimPrinciple[*types.ProjectScope]{
		Info: &types.ProjectScope{
			OrganizationId: &ath.OrganizationId,
			ProjectId:      &ath.ProjectId,
			Status:         ath.GetStatus(),
			CurrentToken:   claimToken,
		},
	}, nil
}
