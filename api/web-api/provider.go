package web_api

import (
	"context"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type webProviderApi struct {
	cfg      *config.AppConfig
	logger   commons.Logger
	postgres connectors.PostgresConnector
	redis    connectors.RedisConnector
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
			cfg:      config,
			logger:   logger,
			postgres: postgres,
			redis:    redis,
		},
	}
}

// GetAllToolProvider implements lexatic_backend.ProviderServiceServer.
func (w *webProviderGRPCApi) GetAllToolProvider(context.Context, *web_api.GetAllToolProviderRequest) (*web_api.GetAllToolProviderResponse, error) {
	panic("unimplemented")
}

// GetAllModel implements lexatic_backend.ProviderServiceServer.
func (w *webProviderGRPCApi) GetAllModel(context.Context, *web_api.GetAllModelRequest) (*web_api.GetAllModelResponse, error) {
	panic("unimplemented")
}

// GetAllProvider implements lexatic_backend.ProviderServiceServer.
func (w *webProviderGRPCApi) GetAllProvider(context.Context, *web_api.GetAllProviderRequest) (*web_api.GetAllProviderResponse, error) {
	panic("unimplemented")
}

// GetModel implements lexatic_backend.ProviderServiceServer.
func (w *webProviderGRPCApi) GetModel(context.Context, *web_api.GetModelRequest) (*web_api.GetModelResponse, error) {
	panic("unimplemented")
}
