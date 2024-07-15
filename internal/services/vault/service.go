package internal_vault_service

import (
	"context"
	"fmt"

	internal_gorm "github.com/lexatic/web-backend/internal/gorm"
	internal_services "github.com/lexatic/web-backend/internal/services"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	gorm_models "github.com/lexatic/web-backend/pkg/models/gorm"
	gorm_types "github.com/lexatic/web-backend/pkg/models/gorm/types"
	"github.com/lexatic/web-backend/pkg/types"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"gorm.io/gorm/clause"
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

func (vS *vaultService) CreateOrganizationToolCredential(ctx context.Context,
	auth types.Principle,
	toolId uint64,
	name string, credential map[string]interface{}) (*internal_gorm.Vault, error) {
	return vS.createCredential(ctx, *auth.GetUserId(), name, credential, gorm_types.VAULT_TYPE_TOOL,
		toolId, gorm_types.VAULT_LEVEL_ORGANIZATION, *auth.GetCurrentOrganizationId())
}

func (vS *vaultService) CreateOrganizationProviderCredential(ctx context.Context,
	auth types.Principle,
	providerId uint64,
	name string, credential map[string]interface{}) (*internal_gorm.Vault, error) {
	return vS.createCredential(ctx, *auth.GetUserId(), name, credential, gorm_types.VAULT_TYPE_PROVIDER,
		providerId, gorm_types.VAULT_LEVEL_ORGANIZATION, *auth.GetCurrentOrganizationId())

}

func (vS *vaultService) CreateUserToolCredential(ctx context.Context,
	auth types.Principle,
	toolId uint64,
	name string,
	credential map[string]interface{},
) (*internal_gorm.Vault, error) {
	return vS.createCredential(ctx, *auth.GetUserId(), name,
		credential,
		gorm_types.VAULT_TYPE_TOOL,
		toolId,
		gorm_types.VAULT_LEVEL_USER,
		*auth.GetUserId())
}

func (vs *vaultService) createCredential(ctx context.Context,
	userId uint64,
	name string,
	credential map[string]interface{},
	vaultType gorm_types.VaultType,
	vaultTypeId uint64, level gorm_types.VaultLevel, levelId uint64) (*internal_gorm.Vault, error) {
	db := vs.postgres.DB(ctx)
	vlt := &internal_gorm.Vault{
		Name:         name,
		VaultType:    vaultType,
		VaultTypeId:  vaultTypeId,
		Value:        credential,
		CreatedBy:    userId,
		VaultLevel:   level,
		VaultLevelId: levelId,
	}

	tx := db.Save(vlt)
	if err := tx.Error; err != nil {
		vs.logger.Debugf("unable to create organization credentials for tool %v", err)
		return nil, err
	}
	return vlt, nil
}

func (vS *vaultService) Delete(ctx context.Context, auth types.Principle, vaultId uint64) (*internal_gorm.Vault, error) {
	db := vS.postgres.DB(ctx)
	vlt := &internal_gorm.Vault{
		Status:    "deleted",
		UpdatedBy: *auth.GetUserId(),
	}
	tx := db.Where("id = ? and vault_level = ? AND vault_level_id = ?", vaultId, gorm_types.VAULT_LEVEL_ORGANIZATION, *auth.GetCurrentOrganizationId()).Clauses(clause.Returning{}).Updates(vlt)
	if err := tx.Error; err != nil {
		vS.logger.Debugf("unable to delete vault %v")
		return nil, err
	}
	return vlt, nil
}

func (vS *vaultService) GetAllOrganizationCredential(ctx context.Context, auth types.Principle, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_gorm.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vaults []internal_gorm.Vault
	var cnt int64

	qry := db.Model(internal_gorm.Vault{})
	qry.
		Where("vault_level = ? AND vault_level_id = ? AND status = ?", gorm_types.VAULT_LEVEL_ORGANIZATION, *auth.GetCurrentOrganizationId(), "active")
	for _, ct := range criterias {
		qry.Where(fmt.Sprintf("%s = ?", ct.GetKey()), ct.GetValue())
	}
	tx := qry.
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					int(paginate.GetPage()),
					int(paginate.GetPageSize()),
					&cnt,
					qry))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).Find(&vaults)

	if tx.Error != nil {
		vS.logger.Debugf("unable to find any vault %v", *auth.GetCurrentOrganizationId())
		return cnt, nil, tx.Error
	}

	return cnt, &vaults, nil
}

// gorm_types.VAULT_TYPE_PROVIDER,
// rapidaProviderId,
func (vS *vaultService) GetProviderCredential(ctx context.Context,
	auth types.SimplePrinciple,
	providerId uint64) (*internal_gorm.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vault internal_gorm.Vault

	tx := db.Where("status = ? and vault_type = ? and vault_type_id = ? and vault_level = ? and vault_level_id = ?",
		"active",
		string(gorm_types.VAULT_TYPE_PROVIDER),
		providerId,
		string(gorm_types.VAULT_LEVEL_ORGANIZATION),
		*auth.GetCurrentOrganizationId(),
	).First(&vault)
	if tx.Error != nil {
		vS.logger.Errorf("get credential error  %v", tx.Error)
		return nil, tx.Error
	}
	return &vault, nil
}

func (vS *vaultService) CreateRapidaProviderCredential(ctx context.Context,
	organizationId uint64) (*internal_gorm.Vault, error) {

	rapidaProviderId := uint64(8298870085084815298)
	return vS.createCredential(
		ctx,
		99,
		"default-rapida-credentials",
		map[string]interface{}{
			"1987967168347635712": map[string]string{
				"key": "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",
			},
			"5212367370329048775": map[string]string{
				"key": "pa-f2E7NuNmZrC9ADFKE6KjnyZgxUH8_xliK3C0CAKVG00",
			},
			"198796716894742120": map[string]string{
				"key": "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",
			},
			"198796716894742119": map[string]string{
				"key": "pskpIUoGzlLvzuxczSpLaJpxuFhZiWzy",
			},
			"198796716894742118": map[string]string{
				"key": "AIzaSyBI19ykmjj-wsA_hR3rt-UrPkFtwQQ3hhY",
			},
			"1987967168452493312": map[string]string{
				"key": "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",
			},
			"1987967168435716096": map[string]string{
				"key": "nHuteTe84dihnImlgpwiD7Tk9cmAHP1qxHocstf5",
			},
			"1987967168431521792": map[string]string{
				"key": "r8_FvPKVcfvtL3NifEKEUvi7q2uEpNYUsm3MUpMN",
			},
		},
		gorm_types.VAULT_TYPE_PROVIDER,
		rapidaProviderId,
		gorm_types.VAULT_LEVEL_ORGANIZATION,
		organizationId,
	)
}
