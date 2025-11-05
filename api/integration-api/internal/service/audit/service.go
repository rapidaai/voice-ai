package internal_audit_service

import (
	"context"
	"fmt"

	internal_gorm "github.com/rapidaai/api/integration-api/internal/entity"
	internal_services "github.com/rapidaai/api/integration-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	integration_api "github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

type auditService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func NewAuditService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.AuditService {
	return &auditService{
		logger:   logger,
		postgres: postgres,
	}
}

// do not play with proto object as mutex
func (aS *auditService) Create(ctx context.Context,
	requestId, organizationId, projectId, credentialId uint64, intName,
	assetPrefix string, mertics types.Metrics, status string) (*internal_gorm.ExternalAudit, error) {
	db := aS.postgres.DB(ctx)

	audit := &internal_gorm.ExternalAudit{
		Audited: gorm_models.Audited{
			Id: requestId,
		},
		OrganizationId:  organizationId,
		ProjectId:       projectId,
		CredentialId:    credentialId,
		IntegrationName: intName,
		AssetPrefix:     assetPrefix,
		Status:          status,
	}

	if len(mertics) > 0 {
		audit.SetMetrics(mertics)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"metrics", "response_status", "status", "time_taken", "updated_date"}),
	}).Create(audit)
	if tx.Error != nil {
		aS.logger.Errorf("unable to insert into audit table %v", tx.Error)
		return nil, tx.Error
	}
	return audit, nil
}

func (aS *auditService) CreateMetadata(c context.Context, auditId uint64, metadata map[string]string) ([]*internal_gorm.ExternalAuditMetadata, error) {
	if len(metadata) > 0 {
		db := aS.postgres.DB(c)
		_metadata := make([]*internal_gorm.ExternalAuditMetadata, 0)
		for k, v := range metadata {
			_metadata = append(_metadata, &internal_gorm.ExternalAuditMetadata{
				ExternalAuditId: auditId,
				Key:             k,
				Value:           v,
			})
		}
		tx := db.Create(&_metadata)
		if tx.Error != nil {
			aS.logger.Errorf("error while updating model parameter %v", tx.Error)
			return nil, tx.Error
		}
		return _metadata, nil
	}
	return nil, nil
}

func (aS *auditService) UpdateMetadata(c context.Context, auditId uint64, metadata map[string]string) ([]*internal_gorm.ExternalAuditMetadata, error) {
	if len(metadata) > 0 {
		db := aS.postgres.DB(c)
		_metadata := make([]*internal_gorm.ExternalAuditMetadata, 0)
		for k, v := range metadata {
			_metadata = append(_metadata, &internal_gorm.ExternalAuditMetadata{
				ExternalAuditId: auditId,
				Key:             k,
				Value:           v,
			})
		}
		tx := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "external_audit_id"}, {Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value"}),
		}).Create(&_metadata)
		if tx.Error != nil {
			aS.logger.Errorf("error while updating model parameter %v", tx.Error)
			return nil, tx.Error
		}
		return _metadata, nil
	}
	return nil, nil
}

// func (aS *auditService) Complete(ctx context.Context, auditId uint64, metrics types.Metrics) error {
// 	db := aS.postgres.DB(ctx)
// 	audit := &internal_gorm.ExternalAudit{
// 		Status: "complete",
// 	}
// 	audit.SetMetrics(metrics)
// 	tx := db.Where("id = ?", auditId).Updates(&audit)
// 	if tx.Error != nil {
// 		aS.logger.Errorf("unable to update into audit table %v", tx.Error)
// 		return tx.Error
// 	}
// 	return nil
// }

func (aS *auditService) GetAll(ctx context.Context, organizationId uint64, projectId uint64, paginate *integration_api.Paginate, ctrs []*integration_api.Criteria) (int64, []*internal_gorm.ExternalAudit, error) {
	db := aS.postgres.DB(ctx)
	var cnt int64
	audits := make([]*internal_gorm.ExternalAudit, 0)
	qry := db.
		Model(internal_gorm.ExternalAudit{}).
		Preload("ExternalAuditMetadatas").
		Where("organization_id = ? AND project_id = ?", organizationId, projectId)

	for _, ct := range ctrs {
		qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
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
		}).
		Find(&audits)
	if tx.Error != nil {
		aS.logger.Errorf("error while quering audit log %v", tx.Error)
		return cnt, nil, tx.Error
	}

	return cnt, audits, nil
}

func (aS *auditService) Get(ctx context.Context, organizationId uint64, projectId uint64, auditId uint64) (*internal_gorm.ExternalAudit, error) {
	db := aS.postgres.DB(ctx)
	audit := &internal_gorm.ExternalAudit{}
	tx := db.
		Model(internal_gorm.ExternalAudit{}).
		Preload("ExternalAuditMetadatas").
		Where("id = ? AND organization_id = ? AND project_id = ?", auditId, organizationId, projectId).
		First(&audit)
	if tx.Error != nil {
		aS.logger.Errorf("error while quering audit log %v", tx.Error)
		return nil, tx.Error
	}

	return audit, nil
}
