package web_api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	config "github.com/rapidaai/api/web-api/config"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	internal_notification_service "github.com/rapidaai/api/web-api/internal/service/notification"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type webNotificationApi struct {
	cfg                 *config.WebAppConfig
	logger              commons.Logger
	postgres            connectors.PostgresConnector
	redis               connectors.RedisConnector
	notificationService internal_service.NotificationService
}

type webNotificationRPCApi struct {
	webNotificationApi
}

type webNotificationGRPCApi struct {
	webNotificationApi
}

func NewNotificationGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.NotificationServiceServer {
	return &webNotificationGRPCApi{
		webNotificationApi{
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			redis:               redis,
			notificationService: internal_notification_service.NewNotificationService(logger, postgres),
		},
	}
}

// GetNotificationSettting implements protos.NotificationServiceServer.
func (nts *webNotificationGRPCApi) GetNotificationSettting(ctx context.Context, ir *protos.GetNotificationSettingRequest) (*protos.NotificationSettingResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	settings, err := nts.notificationService.GetAllNotificationSetting(ctx, principal, principal.Subject().ID)
	if err != nil {
		nts.logger.Errorf("vaultService.GetAll from grpc with err %v", err)
		return utils.Error[protos.NotificationSettingResponse](
			err,
			"Unable to get notification settings, please try again later.",
		)
	}
	out := make([]*protos.NotificationSetting, len(settings))
	err = utils.Cast(settings, &out)
	if err != nil {
		nts.logger.Errorf("unable to cast setting object to proto %v", err)
	}
	return &protos.NotificationSettingResponse{
		Code:    200,
		Success: true,
		Data:    out,
	}, nil
}

// UpdateNotificationSetting implements protos.NotificationServiceServer.
func (nts *webNotificationGRPCApi) UpdateNotificationSetting(ctx context.Context, irequest *protos.UpdateNotificationSettingRequest) (*protos.NotificationSettingResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	_, err := nts.notificationService.UpdateNotificationSetting(ctx, principal, principal.Subject().ID, irequest.GetSettings())
	if err != nil {
		return &protos.NotificationSettingResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to update the organization, please try again in sometime.",
			}}, nil
	}
	// response
	return &protos.NotificationSettingResponse{
		Code:    200,
		Success: true,
	}, nil
}
