package internal_vault_service

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	web_api "github.com/rapidaai/protos"
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

func (vs *vaultService) Create(ctx context.Context,
	auth authentication.Principal,
	provider string,
	name string, credential map[string]interface{}) (*internal_entity.Vault, error) {
	tenant := auth.Tenant()
	if tenant.OrganizationID == 0 || tenant.ProjectID == 0 {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	caller := auth.Caller()
	db := vs.postgres.DB(ctx)
	vlt := &internal_entity.Vault{
		Mutable: gorm_models.Mutable{
			CreatedActorType: string(caller.Type),
			CreatedActorID:   caller.ID,
		},
		Organizational: gorm_models.Organizational{
			OrganizationId: tenant.OrganizationID,
			ProjectId:      tenant.ProjectID,
		},
		Name:     name,
		Provider: provider,
		Value:    credential,
	}

	tx := db.Save(vlt)
	if err := tx.Error; err != nil {
		vs.logger.Debugf("unable to create organization credentials for tool %v", err)
		return nil, err
	}
	return vlt, nil
}

func (vS *vaultService) Delete(ctx context.Context, auth authentication.Principal, vaultId uint64) (*internal_entity.Vault, error) {
	tenant := auth.Tenant()
	if tenant.OrganizationID == 0 || tenant.ProjectID == 0 {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	caller := auth.Caller()
	db := vS.postgres.DB(ctx)
	vlt := &internal_entity.Vault{}
	tx := db.Model(vlt).Where("id = ? AND organization_id = ? AND project_id = ?", vaultId, tenant.OrganizationID, tenant.ProjectID).Clauses(clause.Returning{}).Updates(map[string]interface{}{
		"status":             type_enums.RECORD_ARCHIEVE,
		"updated_actor_type": string(caller.Type),
		"updated_actor_id":   caller.ID,
		"updated_date":       time.Now(),
	})
	if err := tx.Error; err != nil {
		vS.logger.Debugf("unable to delete vault %v", err)
		return nil, err
	}
	return vlt, nil
}

func (vS *vaultService) GetAllOrganizationCredential(ctx context.Context, auth authentication.Principal, criteria []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_entity.Vault, error) {
	tenant := auth.Tenant()
	if tenant.OrganizationID == 0 || tenant.ProjectID == 0 {
		return 0, nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	var vaults []*internal_entity.Vault
	var cnt int64
	err := vS.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&internal_entity.Vault{}).
			Joins("JOIN projects project ON project.id = vaults.project_id AND project.organization_id = vaults.organization_id AND project.status = ?", type_enums.RECORD_ACTIVE).
			Where("vaults.organization_id = ? AND vaults.project_id = ? AND vaults.status = ?", tenant.OrganizationID, tenant.ProjectID, type_enums.RECORD_ACTIVE)
		for _, criterion := range criteria {
			switch criterion.GetKey() {
			case "name":
				pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(criterion.GetValue()) + "%"
				query = query.Where(`LOWER(vaults.name) LIKE LOWER(?) ESCAPE '\'`, pattern)
			case "provider":
				query = query.Where("vaults.provider = ?", criterion.GetValue())
			}
		}
		if err := query.Count(&cnt).Error; err != nil {
			return err
		}
		offset, conversionErr := utils.Uint64ToInt(uint64(paginate.GetPage()-1) * uint64(paginate.GetPageSize()))
		if conversionErr != nil {
			return conversionErr
		}
		return query.Order("vaults.created_date DESC").Order("vaults.id DESC").
			Offset(offset).Limit(int(paginate.GetPageSize())).Find(&vaults).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		vS.logger.Debugf("unable to find any vault for organization %d and project %d", tenant.OrganizationID, tenant.ProjectID)
		return 0, nil, err
	}

	return cnt, vaults, nil
}

func (vS *vaultService) Get(ctx context.Context, tenant authentication.Tenant, id uint64) (*internal_entity.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vault internal_entity.Vault
	tx := db.Where("id = ? AND status = ? AND organization_id = ? AND project_id = ?",
		id,
		type_enums.RECORD_ACTIVE.String(),
		tenant.OrganizationID,
		tenant.ProjectID,
	).Last(&vault)
	if tx.Error != nil {
		vS.logger.Errorf("get credential error  %v", tx.Error)
		return nil, tx.Error
	}
	return &vault, nil
}

func (vS *vaultService) GetProviderCredential(ctx context.Context, organizationID uint64, provider string) (*internal_entity.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vault internal_entity.Vault
	tx := db.Where("provider = ? AND status = ? AND organization_id = ?",
		provider,
		type_enums.RECORD_ACTIVE.String(),
		organizationID,
	).Last(&vault)
	if tx.Error != nil {
		vS.logger.Errorf("get credential error  %v", tx.Error)
		return nil, tx.Error
	}
	return &vault, nil
}
