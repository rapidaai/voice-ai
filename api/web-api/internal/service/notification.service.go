package internal_service

import (
	"context"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/protos"
)

type NotificationService interface {
	UpdateNotificationSetting(ctx context.Context, auth authentication.Principal, userAuthId uint64, settings []*protos.NotificationSetting) ([]*internal_entity.NotificationSetting, error)
	GetAllNotificationSetting(ctx context.Context, auth authentication.Principal, userAuthId uint64) ([]*internal_entity.NotificationSetting, error)
}
