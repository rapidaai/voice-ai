package web_api

import (
	"context"

	config "github.com/lexatic/web-backend/config"
	internal_services "github.com/lexatic/web-backend/internal/services"
	internal_lead_service "github.com/lexatic/web-backend/internal/services/lead"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type webLeadApi struct {
	cfg         *config.AppConfig
	logger      commons.Logger
	postgres    connectors.PostgresConnector
	redis       connectors.RedisConnector
	leadService internal_services.LeadService
}

type webLeadRPCApi struct {
	webLeadApi
}

type webLeadGRPCApi struct {
	webLeadApi
}

func NewLeadRPC(config *config.AppConfig, logger commons.Logger,
	postgres connectors.PostgresConnector, redis connectors.RedisConnector) *webLeadRPCApi {
	return &webLeadRPCApi{
		webLeadApi{
			cfg:         config,
			logger:      logger,
			postgres:    postgres,
			redis:       redis,
			leadService: internal_lead_service.NewLeadService(logger, postgres),
		},
	}
}

func NewLeadGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.LeadServiceServer {
	return &webLeadGRPCApi{
		webLeadApi{
			cfg:         config,
			logger:      logger,
			postgres:    postgres,
			redis:       redis,
			leadService: internal_lead_service.NewLeadService(logger, postgres),
		},
	}
}

func (lS *webLeadGRPCApi) CreateLead(ctx context.Context, irRequest *web_api.LeadCreationRequest) (*web_api.BaseResponse, error) {
	lS.logger.Debugf("CreateLead from grpc with requestPayload %v, %v", irRequest, ctx)
	_, err := lS.leadService.Create(ctx, irRequest.Email)
	if err != nil {
		return nil, err
	}

	return &web_api.BaseResponse{
		Code:    200,
		Success: true,
	}, nil
}
