package web_handler

import (
	"context"
	"fmt"

	internal_service "github.com/rapidaai/api/web-api/internal/service"
	internal_lead_service "github.com/rapidaai/api/web-api/internal/service/lead"
	protos "github.com/rapidaai/protos"

	config "github.com/rapidaai/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
)

type webLeadApi struct {
	WebApi
	cfg         *config.AppConfig
	logger      commons.Logger
	postgres    connectors.PostgresConnector
	redis       connectors.RedisConnector
	leadService internal_service.LeadService
}

type webLeadGRPCApi struct {
	webLeadApi
}

func NewLeadGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.LeadGeneratorServiceServer {
	return &webLeadGRPCApi{
		webLeadApi{
			WebApi:      NewWebApi(config, logger, postgres, redis),
			cfg:         config,
			logger:      logger,
			postgres:    postgres,
			redis:       redis,
			leadService: internal_lead_service.NewLeadService(logger, postgres),
		},
	}
}

func (leadApi *webLeadApi) CreateLead(ctx context.Context, lRequest *protos.CreateLeadRequest) (*protos.BaseResponse, error) {
	ld, err := leadApi.leadService.Create(ctx, lRequest.GetEmail(), lRequest.GetCompany(), lRequest.GetExpectedVolume())
	if err != nil {
		return &protos.BaseResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create lead, please try again",
			},
		}, nil
	}
	return &protos.BaseResponse{
		Code:    200,
		Success: true,
		Data: map[string]string{
			"leadId": fmt.Sprintf("%d", ld.Id),
		},
	}, nil
}
