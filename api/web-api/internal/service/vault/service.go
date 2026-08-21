package internal_vault_service

import (
	"context"
	"fmt"

	"gorm.io/gorm/clause"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
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
	userID uint64,
	projectContext types.ProjectContext,
	provider string,
	name string, credential map[string]interface{}) (*internal_entity.Vault, error) {
	db := vs.postgres.DB(ctx)
	vlt := &internal_entity.Vault{
		Mutable: gorm_models.Mutable{
			CreatedBy: userID,
		},
		Organizational: gorm_models.Organizational{
			OrganizationId: projectContext.OrganizationID,
			ProjectId:      projectContext.ProjectID,
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

func (vS *vaultService) Delete(ctx context.Context, userID uint64, projectContext types.ProjectContext, vaultId uint64) (*internal_entity.Vault, error) {
	db := vS.postgres.DB(ctx)
	vlt := &internal_entity.Vault{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			UpdatedBy: userID,
		},
	}
	tx := db.Where("id = ? AND organization_id = ? AND project_id = ?", vaultId, projectContext.OrganizationID, projectContext.ProjectID).Clauses(clause.Returning{}).Updates(vlt)
	if err := tx.Error; err != nil {
		vS.logger.Debugf("unable to delete vault %v")
		return nil, err
	}
	return vlt, nil
}

func (vS *vaultService) GetAllOrganizationCredential(ctx context.Context, projectContext types.ProjectContext, criteria []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_entity.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vaults []*internal_entity.Vault
	var cnt int64

	qry := db.Model(internal_entity.Vault{})
	qry.
		Where("organization_id = ? AND project_id = ? AND status = ?",
			projectContext.OrganizationID,
			projectContext.ProjectID, type_enums.RECORD_ACTIVE)
	for _, ct := range criteria {
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
					paginate.GetPage(),
					paginate.GetPageSize(),
					&cnt,
					qry))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).Find(&vaults)

	if tx.Error != nil {
		vS.logger.Debugf("unable to find any vault %v", projectContext.OrganizationID)
		return cnt, nil, tx.Error
	}

	return cnt, vaults, nil
}

func (vS *vaultService) Get(ctx context.Context, projectContext types.ProjectContext, id uint64) (*internal_entity.Vault, error) {
	db := vS.postgres.DB(ctx)
	var vault internal_entity.Vault
	tx := db.Where("id = ? AND status = ? AND organization_id = ? AND project_id = ?",
		id,
		type_enums.RECORD_ACTIVE.String(),
		projectContext.OrganizationID,
		projectContext.ProjectID,
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
