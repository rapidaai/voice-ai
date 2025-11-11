package authenticators

import (
	"context"
	"time"

	"github.com/rapidaai/config"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

type userAuthenticator struct {
	logger     commons.Logger
	cfg        *config.AppConfig
	authClient web_client.AuthClient
}

func NewUserAuthenticator(cfg *config.AppConfig, logger commons.Logger, authClient web_client.AuthClient) types.Authenticator {
	return &userAuthenticator{
		logger: logger, cfg: cfg, authClient: authClient,
	}
}

// AuthPrinciple implements types.Authenticator.
func (*userAuthenticator) AuthPrinciple(ctx context.Context, userId uint64) (types.Principle, error) {
	panic("unimplemented")
}
func (authenticator *userAuthenticator) Authorize(ctx context.Context, authToken string, userId uint64) (types.Principle, error) {
	start := time.Now()
	ath, err := authenticator.authClient.Authorize(ctx, authToken, userId)
	if err != nil {
		return nil, err
	}

	roles := ath.GetProjectRoles()
	_rls := make([]*types.ProjectRole, 0)
	for _, r := range roles {
		_rls = append(_rls, &types.ProjectRole{
			Id:          r.GetId(),
			ProjectId:   r.GetProjectId(),
			Role:        r.GetRole(),
			ProjectName: r.GetProjectName(),
		})
	}
	authenticator.logger.Debugf("Benchmarking: userAuthenticator.Authorize time taken %v", time.Since(start))
	return &types.PlainAuthPrinciple{
		User: types.UserInfo{
			Id:    ath.GetUser().GetId(),
			Name:  ath.GetUser().GetName(),
			Email: ath.GetUser().GetEmail(),
		},
		Token: types.AuthToken{
			Id:        ath.GetToken().GetId(),
			Token:     ath.GetToken().GetToken(),
			TokenType: ath.GetToken().GetTokenType(),
			IsExpired: ath.GetToken().GetIsExpired(),
		},
		OrganizationRole: &types.OrganizaitonRole{
			Id:               ath.GetOrganizationRole().GetId(),
			OrganizationId:   ath.GetOrganizationRole().GetOrganizationId(),
			Role:             ath.GetOrganizationRole().GetRole(),
			OrganizationName: ath.GetOrganizationRole().GetOrganizationName(),
		},
		ProjectRoles: _rls,
		CurrentToken: authToken,
	}, nil

}
