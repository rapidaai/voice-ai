package internal_organization_service

import (
	"context"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

func NewOrganizationService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.OrganizationService {
	return &organizationService{
		logger:   logger,
		postgres: postgres,
	}
}

type organizationService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func (oS *organizationService) Create(ctx context.Context, auth *types.Authentication, name string, size string, industry string) (*internal_entity.Organization, error) {
	userContext, err := auth.UserContext()
	if err != nil {
		return nil, err
	}
	db := oS.postgres.DB(ctx)
	org := &internal_entity.Organization{
		Name:     name,
		Industry: industry,
		Size:     size,
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: userContext.UserID,
		},
	}
	tx := db.Save(org)
	if err := tx.Error; err != nil {
		return nil, err
	} else {
		return org, nil
	}
}

func (oS *organizationService) Get(ctx context.Context, organizationId uint64) (*internal_entity.Organization, error) {
	db := oS.postgres.DB(ctx)
	var ct internal_entity.Organization
	tx := db.Last(&ct, "id = ?", organizationId)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &ct, nil
}

func (oS *organizationService) Update(ctx context.Context, auth *types.Authentication, organizationId uint64, name *string, industry *string, email *string) (*internal_entity.Organization, error) {
	userContext, err := auth.UserContext()
	if err != nil {
		return nil, err
	}
	db := oS.postgres.DB(ctx)
	org := &internal_entity.Organization{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			UpdatedBy: userContext.UserID,
		},
	}

	if name != nil {
		org.Name = *name
	}
	if industry != nil {
		org.Industry = *industry
	}
	if email != nil {
		org.Contact = *email
	}
	tx := db.Where("id = ? ", organizationId).Updates(org)
	if err := tx.Error; err != nil {
		return nil, err
	} else {
		return org, nil
	}
}
