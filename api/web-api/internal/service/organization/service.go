package internal_organization_service

import (
	"context"

	internal_entity "github.com/lexatic/web-backend/api/web-api/internal/entity"
	internal_services "github.com/lexatic/web-backend/api/web-api/internal/service"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	gorm_models "github.com/lexatic/web-backend/pkg/models/gorm"
	"github.com/lexatic/web-backend/pkg/types"
	type_enums "github.com/lexatic/web-backend/pkg/types/enums"
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

func (oS *organizationService) Create(ctx context.Context, auth types.Principle, name string, size string, industry string) (*internal_entity.Organization, error) {
	db := oS.postgres.DB(ctx)
	org := &internal_entity.Organization{
		Name:     name,
		Industry: industry,
		Size:     size,
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: auth.GetUserInfo().Id,
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

func (oS *organizationService) Update(ctx context.Context, auth types.Principle, organizationId uint64, name *string, industry *string, email *string) (*internal_entity.Organization, error) {
	db := oS.postgres.DB(ctx)
	org := &internal_entity.Organization{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			UpdatedBy: auth.GetUserInfo().Id,
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

// func (oS *organizationService) InviteUserToOrg(ctx context.Context, userId uint64, role string, auth types.Principle) (*internal_entity.UserOrganizationRole, error) {
// 	db := oS.postgres.DB(ctx)
// 	var org internal_entity.Organization
// 	tx := db.First(&org, "id = ?", auth.GetOrganizationRole().OrganizationId)
// 	if tx.Error != nil {
// 		return nil, tx.Error
// 	}

// 	orgRole := &internal_entity.UserOrganizationRole{
// 		UserAuthId:     userId,
// 		Role:           strings.ToLower(role),
// 		OrganizationId: auth.GetOrganizationRole().OrganizationId,
// 		Status:         "invited",
// 		CreatedBy:      auth.GetUserInfo().Id,
// 		UpdatedBy:      auth.GetUserInfo().Id,
// 	}

// 	if err := db.Save(orgRole).Error; err != nil {
// 		return nil, err
// 	}
// 	return orgRole, nil
// }

// func (oS *organizationService) GetRoles(ctx context.Context, organizationId uint64) ([]*internal_entity.UserOrganizationRole, error) {
// 	db := oS.postgres.DB(ctx)
// 	var roles []*internal_entity.UserOrganizationRole
// 	if err := db.Where("organization_id = ? and status = ?", organizationId, "active").Find(&roles).Error; err != nil {
// 		return nil, err
// 	}

// 	return roles, nil
// }
