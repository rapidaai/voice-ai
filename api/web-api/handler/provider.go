package web_handler

import (
	"context"

	internal_service "github.com/lexatic/web-backend/api/web-api/internal/service"
	internal_provider_service "github.com/lexatic/web-backend/api/web-api/internal/service/provider"
	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type webProviderApi struct {
	WebApi
	cfg             *config.AppConfig
	logger          commons.Logger
	postgres        connectors.PostgresConnector
	redis           connectors.RedisConnector
	providerService internal_service.ProviderService
}

type webProviderRPCApi struct {
	webProviderApi
}

type webProviderGRPCApi struct {
	webProviderApi
}

func NewProviderGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.ProviderServiceServer {
	return &webProviderGRPCApi{
		webProviderApi{
			WebApi:          NewWebApi(config, logger, postgres, redis),
			cfg:             config,
			logger:          logger,
			postgres:        postgres,
			redis:           redis,
			providerService: internal_provider_service.NewProviderService(logger, postgres),
		},
	}
}

// GetAllModel implements lexatic_backend.ProviderServiceServer.
func (w *webProviderGRPCApi) GetAllModel(ctx context.Context, r *web_api.GetAllModelRequest) (*web_api.GetAllModelResponse, error) {
	models, err := w.providerService.GetAllModel(ctx, r.GetCriterias())
	if err != nil {
		w.logger.Errorf("error while getting all the model with %+v type err %v", r.GetCriterias(), err)
		return utils.Error[web_api.GetAllModelResponse](err, "Unable to get all the models, please try again.")
	}

	var mdls = []*web_api.ProviderModel{}
	err = utils.Cast(models, &mdls)
	if err != nil {
		w.logger.Errorf("error while type casting model type err %v", err)
	}
	return utils.PaginatedSuccess[web_api.GetAllModelResponse, []*web_api.ProviderModel](
		uint32(len(mdls)), 0,
		mdls)
}

// GetModel implements lexatic_backend.ProviderServiceServer.
func (w *webProviderGRPCApi) GetModel(ctx context.Context, r *web_api.GetModelRequest) (*web_api.GetModelResponse, error) {
	provider, err := w.providerService.GetModel(ctx, r.ModelId)
	if err != nil {
		return utils.Error[web_api.GetModelResponse](err, "Unable to get the models, please try again.")
	}
	model := &web_api.ProviderModel{}
	err = utils.Cast(provider, model)
	if err != nil {
		w.logger.Debugf("error while type casting model type err %v", err)
	}
	return utils.Success[web_api.GetModelResponse](model)
}

// GetAllProvider implements lexatic_backend.ProviderServiceServer.
func (w *webProviderGRPCApi) GetAllModelProvider(ctx context.Context, gat *web_api.GetAllModelProviderRequest) (*web_api.GetAllModelProviderResponse, error) {
	providers, err := w.providerService.GetAllModelProvider(ctx, gat.GetCriterias())
	if err != nil {
		return utils.Error[web_api.GetAllModelProviderResponse](
			err,
			"Unable to get tool providers, please try again in sometime.")
	}
	prds := []*web_api.Provider{}
	err = utils.Cast(providers, &prds)
	if err != nil {
		w.logger.Errorf("unable to cast tool provider to proto object %v", err)
	}
	return utils.PaginatedSuccess[web_api.GetAllModelProviderResponse, []*web_api.Provider](
		uint32(len(prds)), 0,
		prds)
}

func (w *webProviderGRPCApi) GetAllToolProvider(ctx context.Context, gat *web_api.GetAllToolProviderRequest) (*web_api.GetAllToolProviderResponse, error) {
	providers, err := w.providerService.GetAllToolProvider(ctx, gat.GetCriterias())
	if err != nil {
		return utils.Error[web_api.GetAllToolProviderResponse](
			err,
			"Unable to get tool providers, please try again in sometime.")
	}
	prds := []*web_api.ToolProvider{}
	err = utils.Cast(providers, &prds)
	if err != nil {
		w.logger.Errorf("unable to cast tool provider to proto object %v", err)
	}
	return utils.PaginatedSuccess[web_api.GetAllToolProviderResponse, []*web_api.ToolProvider](
		uint32(len(prds)), 1,
		prds)
}
