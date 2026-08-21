package internal_notification_service

import (
	"context"

	"gorm.io/gorm/clause"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
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

func (oS *notificationService) UpdateNotificationSetting(ctx context.Context, auth *types.Authentication, authId uint64, settings []*protos.NotificationSetting) ([]*internal_entity.NotificationSetting, error) {
	userContext, err := auth.UserContext()
	if err != nil {
		return nil, err
	}
	db := oS.postgres.DB(ctx)
	nts := make([]*internal_entity.NotificationSetting, 0)
	for _, st := range settings {
		nts = append(nts, &internal_entity.NotificationSetting{
			EventType:  st.GetEventType(),
			Channel:    st.GetChannel(),
			Enabled:    st.GetEnabled(),
			UserAuthId: authId,
			Mutable: gorm_models.Mutable{
				Status:    type_enums.RECORD_ACTIVE,
				CreatedBy: userContext.UserID,
			},
		})
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "channel"}, {Name: "event_type"}, {Name: "user_auth_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"enabled",
			"updated_by", "updated_date"}),
	}).Save(nts)
	if err := tx.Error; err != nil {
		return nil, err
	} else {
		return nts, nil
	}
}

func (oS *notificationService) GetAllNotificationSetting(ctx context.Context, auth *types.Authentication, userId uint64) ([]*internal_entity.NotificationSetting, error) {
	db := oS.postgres.DB(ctx)
	var nts []*internal_entity.NotificationSetting
	tx := db.Where("user_auth_id = ?", userId).Find(&nts)
	if err := tx.Error; err != nil {
		return nts, err
	}
	return nts, nil
}
