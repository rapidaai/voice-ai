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

func (vS *vaultService) CreateAllDefaultKeys(ctx context.Context, organizationId uint64) ([]*internal_gorm.Vault, error) {
	db := vS.postgres.DB(ctx)
	vlts := make([]*internal_gorm.Vault, 0)

	vlts = append(vlts,
		&internal_gorm.Vault{
			Name:           "default-anthropic-01",
			ProviderId:     1987967168347635712,
			Key:            "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",
			CreatedBy:      99,
			OrganizationId: organizationId,
		})
	vlts = append(vlts,
		&internal_gorm.Vault{
			Name:           "default-replicate-01",
			ProviderId:     1987967168431521792,
			Key:            "r8_FvPKVcfvtL3NifEKEUvi7q2uEpNYUsm3MUpMN",
			CreatedBy:      99,
			OrganizationId: organizationId,
		})
	vlts = append(vlts,
		&internal_gorm.Vault{
			Name:           "default-cohere-01",
			ProviderId:     1987967168435716096,
			Key:            "nHuteTe84dihnImlgpwiD7Tk9cmAHP1qxHocstf5",
			CreatedBy:      99,
			OrganizationId: organizationId,
		})
	vlts = append(vlts,
		&internal_gorm.Vault{
			Name:           "default-openai-01",
			ProviderId:     1987967168452493312,
			Key:            "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",
			CreatedBy:      99,
			OrganizationId: organizationId,
		})

	vlts = append(vlts, &internal_gorm.Vault{
		Name:           "default-google-01",
		ProviderId:     198796716894742118,
		Key:            "AIzaSyBI19ykmjj-wsA_hR3rt-UrPkFtwQQ3hhY",
		CreatedBy:      99,
		OrganizationId: organizationId,
	})
	tx := db.Save(vlts)
	if err := tx.Error; err != nil {
		vS.logger.Debugf("unable to insert for default keys of provider %v", err)
		return nil, err
	}
	return vlts, nil
}
