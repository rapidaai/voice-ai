package internal_service

import (
	"context"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type NotificationService interface {
	UpdateNotificationSetting(ctx context.Context, auth *types.Authentication, userAuthId uint64, settings []*protos.NotificationSetting) ([]*internal_entity.NotificationSetting, error)
	GetAllNotificationSetting(ctx context.Context, auth *types.Authentication, userAuthId uint64) ([]*internal_entity.NotificationSetting, error)
}
