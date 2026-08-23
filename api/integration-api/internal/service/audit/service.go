// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_audit_service

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	internal_gorm "github.com/rapidaai/api/integration-api/internal/entity"
	internal_services "github.com/rapidaai/api/integration-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	integration_api "github.com/rapidaai/protos"
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
	assetPrefix string, mertics []*protos.Metric, status type_enums.RecordState) (*internal_gorm.ExternalAudit, error) {
	auth := ctx.Value(types.CTX_).(*types.Authentication)
	db := aS.postgres.DB(ctx)

	audit := &internal_gorm.ExternalAudit{
		Audited: gorm_models.Audited{
			Id: requestId,
		},
		ActorAudit: gorm_models.ActorAudit{
			CreatedActorType: auth.Actor().Type.String(),
			CreatedActorID:   auth.Actor().ID,
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

	assignments := clause.AssignmentColumns([]string{"metrics", "response_status", "status", "time_taken", "updated_date"})
	assignments = append(assignments,
		clause.Assignment{Column: clause.Column{Name: "updated_actor_type"}, Value: auth.Actor().Type.String()},
		clause.Assignment{Column: clause.Column{Name: "updated_actor_id"}, Value: auth.Actor().ID},
	)
	tx := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: assignments,
	}).Create(audit)
	if tx.Error != nil {
		aS.logger.Errorf("unable to insert into audit table %v", tx.Error)
		return nil, tx.Error
	}
	return audit, nil
}

func (aS *auditService) CreateMetadata(c context.Context, auditId uint64, metadata map[string]string) ([]*internal_gorm.ExternalAuditMetadata, error) {
	if len(metadata) > 0 {
		auth := c.Value(types.CTX_).(*types.Authentication)
		db := aS.postgres.DB(c)
		_metadata := make([]*internal_gorm.ExternalAuditMetadata, 0)
		for k, v := range metadata {
			_metadata = append(_metadata, &internal_gorm.ExternalAuditMetadata{
				ExternalAuditId: auditId,
				Key:             k,
				Value:           v,
				ActorAudit: gorm_models.ActorAudit{
					CreatedActorType: auth.Actor().Type.String(),
					CreatedActorID:   auth.Actor().ID,
				},
			})
		}
		assignments := clause.AssignmentColumns([]string{"value", "updated_date"})
		assignments = append(assignments,
			clause.Assignment{Column: clause.Column{Name: "updated_actor_type"}, Value: auth.Actor().Type.String()},
			clause.Assignment{Column: clause.Column{Name: "updated_actor_id"}, Value: auth.Actor().ID},
		)
		tx := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "external_audit_id"}, {Name: "key"}},
			DoUpdates: assignments,
		}).Create(&_metadata)
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
		auth := c.Value(types.CTX_).(*types.Authentication)
		db := aS.postgres.DB(c)
		_metadata := make([]*internal_gorm.ExternalAuditMetadata, 0)
		for k, v := range metadata {
			_metadata = append(_metadata, &internal_gorm.ExternalAuditMetadata{
				ExternalAuditId: auditId,
				Key:             k,
				Value:           v,
				ActorAudit: gorm_models.ActorAudit{
					CreatedActorType: auth.Actor().Type.String(),
					CreatedActorID:   auth.Actor().ID,
				},
			})
		}
		tx := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "external_audit_id"}, {Name: "key"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"value":              clause.Expr{SQL: "excluded.value"},
				"updated_date":       time.Now(),
				"updated_actor_type": auth.Actor().Type.String(),
				"updated_actor_id":   auth.Actor().ID,
			}),
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
		if ct == nil || ct.GetKey() == "" || ct.GetValue() == "" {
			continue
		}

		switch ct.GetKey() {
		case "id":
			qry = qry.Where("external_audits.id = ?", ct.GetValue())
		case "created_date":
			switch ct.GetLogic() {
			case ">=":
				qry = qry.Where("external_audits.created_date >= ?", ct.GetValue())
			case "<=":
				qry = qry.Where("external_audits.created_date <= ?", ct.GetValue())
			case ">":
				qry = qry.Where("external_audits.created_date > ?", ct.GetValue())
			case "<":
				qry = qry.Where("external_audits.created_date < ?", ct.GetValue())
			case "=":
				qry = qry.Where("external_audits.created_date = ?", ct.GetValue())
			}
		case "provider_name", "model_name", "assistant_id", "endpoint_id", "knowledge_id", "source":
			switch ct.GetLogic() {
			case "contains":
				qry = qry.Where(
					"EXISTS (SELECT 1 FROM external_audit_metadata WHERE external_audit_metadata.external_audit_id = external_audits.id AND external_audit_metadata.key = ? AND external_audit_metadata.value ILIKE ?)",
					ct.GetKey(),
					"%"+ct.GetValue()+"%",
				)
			case "=":
				qry = qry.Where(
					"EXISTS (SELECT 1 FROM external_audit_metadata WHERE external_audit_metadata.external_audit_id = external_audits.id AND external_audit_metadata.key = ? AND external_audit_metadata.value = ?)",
					ct.GetKey(),
					ct.GetValue(),
				)
			}
		case "response_status":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("external_audits.response_status = ?", ct.GetValue())
			}
		case "time_taken":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("external_audits.time_taken = ?", ct.GetValue())
			case ">=":
				qry = qry.Where("external_audits.time_taken >= ?", ct.GetValue())
			case "<=":
				qry = qry.Where("external_audits.time_taken <= ?", ct.GetValue())
			}
		case "time_to_first_token":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where(
					"EXISTS (SELECT 1 FROM jsonb_array_elements(external_audits.metrics::jsonb) AS metric(value) WHERE metric.value->>'name' = ? AND (metric.value->>'value')::numeric = ?::numeric)",
					ct.GetKey(),
					ct.GetValue(),
				)
			case ">=":
				qry = qry.Where(
					"EXISTS (SELECT 1 FROM jsonb_array_elements(external_audits.metrics::jsonb) AS metric(value) WHERE metric.value->>'name' = ? AND (metric.value->>'value')::numeric >= ?::numeric)",
					ct.GetKey(),
					ct.GetValue(),
				)
			case "<=":
				qry = qry.Where(
					"EXISTS (SELECT 1 FROM jsonb_array_elements(external_audits.metrics::jsonb) AS metric(value) WHERE metric.value->>'name' = ? AND (metric.value->>'value')::numeric <= ?::numeric)",
					ct.GetKey(),
					ct.GetValue(),
				)
			}
		case "status":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("external_audits.status = ?", ct.GetValue())
			}
		case "integration_name":
			switch ct.GetLogic() {
			case "contains":
				qry = qry.Where("external_audits.integration_name ILIKE ?", "%"+ct.GetValue()+"%")
			case "=":
				qry = qry.Where("external_audits.integration_name = ?", ct.GetValue())
			}
		case "asset_prefix":
			switch ct.GetLogic() {
			case "contains":
				qry = qry.Where("external_audits.asset_prefix ILIKE ?", "%"+ct.GetValue()+"%")
			case "=":
				qry = qry.Where("external_audits.asset_prefix = ?", ct.GetValue())
			}
		case "credential_id":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("external_audits.credential_id = ?", ct.GetValue())
			}
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
