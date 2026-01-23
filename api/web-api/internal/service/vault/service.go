package internal_vault_service

import (
	"context"
	"fmt"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	web_api "github.com/rapidaai/protos"
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

func (vs *vaultService) Create(ctx context.Context,
	auth types.SimplePrinciple,
	provider string,
	name string, credential map[string]interface{}) (*internal_entity.Vault, error) {

	db := vs.postgres.DB(ctx)
	vlt := &internal_entity.Vault{
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
		},
		Organizational: gorm_models.Organizational{
			OrganizationId: *auth.GetCurrentOrganizationId(),
			ProjectId:      *auth.GetCurrentProjectId(),
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

func (vS *vaultService) Delete(ctx context.Context, auth types.Principle, vaultId uint64) (*internal_entity.Vault, error) {
	db := vS.postgres.DB(ctx)
	vlt := &internal_entity.Vault{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIVE,
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("id = ? AND organization_id = ? AND project_id = ?", vaultId, *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId()).Clauses(clause.Returning{}).Updates(vlt)
	if err := tx.Error; err != nil {
		vS.logger.Debugf("unable to delete vault %v")
		return nil, err
	}
	return vlt, nil
}

func (vS *vaultService) GetAllOrganizationCredential(ctx context.Context, auth types.SimplePrinciple, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_entity.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vaults []*internal_entity.Vault
	var cnt int64

	qry := db.Model(internal_entity.Vault{})
	qry.
		Where("organization_id = ? AND project_id = ? AND status = ?",
			*auth.GetCurrentOrganizationId(),
			*auth.GetCurrentProjectId(), type_enums.RECORD_ACTIVE)
	for _, ct := range criterias {
		switch ct.GetLogic() {
		case "or":
			qry.Or(fmt.Sprintf("%s = ?", ct.GetKey()), ct.GetValue())
		case "like":
			qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), fmt.Sprintf("%%%s%%", ct.GetValue()))
		default:
			qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
		}
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

	return cnt, vaults, nil
}

func (vS *vaultService) Get(ctx context.Context, auth types.SimplePrinciple, id uint64) (*internal_entity.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vault internal_entity.Vault
	tx := db.Where("id = ? AND status = ? AND organization_id = ? AND project_id = ?",
		id,
		type_enums.RECORD_ACTIVE.String(),
		*auth.GetCurrentOrganizationId(),
		*auth.GetCurrentProjectId(),
	).Last(&vault)
	if tx.Error != nil {
		vS.logger.Errorf("get credential error  %v", tx.Error)
		return nil, tx.Error
	}
	return &vault, nil
}

func (vS *vaultService) GetProviderCredential(ctx context.Context, auth types.SimplePrinciple, provider string) (*internal_entity.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vault internal_entity.Vault
	tx := db.Where("provider = ? AND status = ? AND organization_id = ?",
		provider,
		type_enums.RECORD_ACTIVE.String(),
		*auth.GetCurrentOrganizationId(),
	).Last(&vault)
	if tx.Error != nil {
		vS.logger.Errorf("get credential error  %v", tx.Error)
		return nil, tx.Error
	}
	return &vault, nil
}
