package internal_notification_service

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

func NewNotificationService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.NotificationService {
	return &notificationService{
		logger:   logger,
		postgres: postgres,
	}
}

type notificationService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func (oS *notificationService) UpdateNotificationSetting(ctx context.Context, auth authentication.Principal, authId uint64, settings []*protos.NotificationSetting) ([]*internal_entity.NotificationSetting, error) {
	subject := auth.Subject()
	if subject.Type != authentication.IdentityUser || subject.ID != authId {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	caller := auth.Caller()
	db := oS.postgres.DB(ctx)
	nts := make([]*internal_entity.NotificationSetting, 0)
	for _, st := range settings {
		nts = append(nts, &internal_entity.NotificationSetting{
			EventType:  st.GetEventType(),
			Channel:    st.GetChannel(),
			Enabled:    st.GetEnabled(),
			UserAuthId: authId,
			Mutable: gorm_models.Mutable{
				Status:           type_enums.RECORD_ACTIVE,
				CreatedActorType: string(caller.Type),
				CreatedActorID:   caller.ID,
			},
		})
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "channel"}, {Name: "event_type"}, {Name: "user_auth_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"enabled":            clause.Expr{SQL: "excluded.enabled"},
			"updated_actor_type": string(caller.Type),
			"updated_actor_id":   caller.ID,
			"updated_date":       time.Now(),
		}),
	}).Save(nts)
	if err := tx.Error; err != nil {
		return nil, err
	} else {
		return nts, nil
	}
}

func (oS *notificationService) GetAllNotificationSetting(ctx context.Context, auth authentication.Principal, userId uint64) ([]*internal_entity.NotificationSetting, error) {
	subject := auth.Subject()
	if subject.Type != authentication.IdentityUser || subject.ID != userId {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	db := oS.postgres.DB(ctx)
	var nts []*internal_entity.NotificationSetting
	tx := db.Where("user_auth_id = ?", userId).Find(&nts)
	if err := tx.Error; err != nil {
		return nts, err
	}
	return nts, nil
}
