package web_api

import (
	"context"

	config "github.com/lexatic/web-backend/config"
	internal_services "github.com/lexatic/web-backend/internal/services"
	internal_user_service "github.com/lexatic/web-backend/internal/services/user"
	provider_client "github.com/lexatic/web-backend/pkg/clients/provider"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type WebApi struct {
	cfg            *config.AppConfig
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	redis          connectors.RedisConnector
	userService    internal_services.UserService
	providerClient provider_client.ProviderServiceClient
}

func NewWebApi(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) WebApi {
	return WebApi{
		cfg, logger, postgres, redis,
		internal_user_service.NewUserService(logger, postgres),
		provider_client.NewProviderServiceClientGRPC(cfg, logger, redis),
	}
}

func (w *WebApi) GetUser(c context.Context, auth types.SimplePrinciple, userId uint64) *web_api.User {
	usr, err := w.userService.GetUser(c, userId)
	if err != nil {
		w.logger.Errorf("unable to get user form the database %+v", err)
		return nil
	}
	ot := &web_api.User{}
	err = utils.Cast(usr, ot)
	if err != nil {
		w.logger.Errorf("unable to cast project to proto object %v", err)
	}
	return ot
}

func (w *WebApi) GetProviderModel(ctx context.Context, auth types.SimplePrinciple, providerModelId uint64) *web_api.ProviderModel {
	mdl, err := w.providerClient.GetModel(ctx, auth, providerModelId)
	if err != nil {
		w.logger.Errorf("unable to get provider model %v", err)
	}
	return mdl
}
