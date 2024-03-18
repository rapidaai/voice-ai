package authenticators

import (
	"context"
	"time"

	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type projectAuthenticator struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
	cfg      *config.AppConfig
}

func NewProjectAuthenticator(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) types.ClaimAuthenticator[*types.ProjectScope] {
	return &projectAuthenticator{
		logger: logger, postgres: postgres, cfg: cfg,
	}
}

func (authenticator *projectAuthenticator) Claim(ctx context.Context, claimToken string) (*types.PlainClaimPrinciple[*types.ProjectScope], error) {
	start := time.Now()
	db := authenticator.postgres.DB(ctx)
	var prjScope *types.ProjectScope
	tx := db.Table("project_credentials").Order("created_date DESC").Where("key = ?", claimToken).First(&prjScope)
	if tx.Error != nil {
		authenticator.logger.Errorf("Authentication error, illegal key request %v", tx.Error)
		return nil, tx.Error
	}
	authenticator.logger.Debugf("Benchmarking: projectAuthenticator.Claim time taken %v", time.Since(start))
	return &types.PlainClaimPrinciple[*types.ProjectScope]{
		Info: prjScope,
	}, nil
}
