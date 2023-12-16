package internal_vault_service

import (
	"context"
	"strings"

	internal_gorm "github.com/lexatic/web-backend/internal/gorm"
	internal_services "github.com/lexatic/web-backend/internal/services"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	gorm_models "github.com/lexatic/web-backend/pkg/models/gorm"
	"github.com/lexatic/web-backend/pkg/types"
)

type vaultService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func NewVaultService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.VaultService {
	return &vaultService{
		logger:   logger,
		postgres: postgres,
	}
}

func (vS *vaultService) Create(ctx context.Context, auth types.Principle, organizationId uint64, providerId uint64, keyName string, key string) (*internal_gorm.Vault, error) {
	db := vS.postgres.DB(ctx)
	vlt := &internal_gorm.Vault{
		Name:           keyName,
		ProviderId:     providerId,
		Key:            key,
		CreatedBy:      auth.GetUserInfo().Id,
		OrganizationId: organizationId,
	}
	tx := db.Save(vlt)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return vlt, nil

}
func (vS *vaultService) Delete(ctx context.Context, auth types.Principle, vaultId uint64) (*internal_gorm.Vault, error) {
	db := vS.postgres.DB(ctx)
	vlt := &internal_gorm.Vault{
		Audited:   gorm_models.Audited{Id: vaultId},
		Status:    "deleted",
		UpdatedBy: auth.GetUserInfo().Id,
	}
	tx := db.Save(vlt)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return vlt, nil
}

func (vS *vaultService) Update(ctx context.Context, auth types.Principle, vaultId uint64, providerId uint64, value string, name string) (*internal_gorm.Vault, error) {
	db := vS.postgres.DB(ctx)
	vlt := &internal_gorm.Vault{
		Audited: gorm_models.Audited{
			Id: vaultId,
		},
		UpdatedBy:  auth.GetUserInfo().Id,
		Name:       name,
		ProviderId: providerId,
	}
	updates := map[string]interface{}{"updated_by": auth.GetUserInfo().Id, "name": name, "provider_id": providerId}
	if strings.TrimSpace(value) != "" {
		updates["key"] = value
	}
	tx := db.Model(&vlt).Updates(updates)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return vlt, nil
}

func (vS *vaultService) GetAll(ctx context.Context, auth types.Principle, organizationId uint64) (*[]internal_gorm.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vaults []internal_gorm.Vault
	tx := db.Where("organization_id = ? AND status = ?", organizationId, "active").Find(&vaults)
	if tx.Error != nil {
		vS.logger.Debugf("unable to find any vault %s", organizationId)
		return nil, tx.Error
	}
	return &vaults, nil
}

func (vS *vaultService) Get(ctx context.Context, organizationId uint64, providerId uint64) (*internal_gorm.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vault internal_gorm.Vault
	if err := db.Where("organization_id = ? and status = ? and provider_id = ?", organizationId, "active", providerId).Find(&vault).Error; err != nil {
		vS.logger.Errorf("get credential error  %v", err)
		return nil, err
	}
	return &vault, nil
}
