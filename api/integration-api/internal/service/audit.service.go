package internal_services

import (
	"context"

	internal_gorm "github.com/rapidaai/api/integration-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

type AuditService interface {
	Get(ctx context.Context, organizationId uint64, projectId uint64, auditId uint64) (*internal_gorm.ExternalAudit, error)
	GetAll(ctx context.Context, organizationId uint64, projectId uint64, paginate *protos.Paginate, opts []*protos.Criteria) (int64, []*internal_gorm.ExternalAudit, error)
	Create(ctx context.Context, requestId, organizationId, projectId, credentialId uint64, intName string, assetPrefix string, metrics types.Metrics, status string) (*internal_gorm.ExternalAudit, error)
	CreateMetadata(c context.Context, auditId uint64, metadata map[string]string) ([]*internal_gorm.ExternalAuditMetadata, error)
	UpdateMetadata(c context.Context, auditId uint64, metadata map[string]string) ([]*internal_gorm.ExternalAuditMetadata, error)
}
