package internal_service

import (
	"context"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
	lexatic_backend "github.com/rapidaai/protos"
)

type EndpointLogService interface {
	CreateEndpointLog(
		ctx context.Context,
		auth types.SimplePrinciple,
		source utils.RapidaSource,
		endpointId, endpointProviderModelId uint64,
		logId uint64,
		arguments, metadata, options map[string]interface{},
	) (*internal_gorm.EndpointLog, error)
	UpdateEndpointLog(
		ctx context.Context,
		auth types.SimplePrinciple,
		logId uint64,
		metrics []*lexatic_backend.Metric,
		timeTaken uint64,
	) (*internal_gorm.EndpointLog, error)

	GetAllEndpointLog(ctx context.Context,
		auth types.SimplePrinciple,
		endpointId uint64,
		criterias []*endpoint_grpc_api.Criteria, paginate *endpoint_grpc_api.Paginate) (int64, []*internal_gorm.EndpointLog, error)
	GetEndpointLog(ctx context.Context, auth types.SimplePrinciple, logId, endpointId uint64) (*internal_gorm.EndpointLog, error)
	GetAggregatedEndpointAnalytics(ctx context.Context, auth types.SimplePrinciple, endpointId uint64) *lexatic_backend.AggregatedEndpointAnalytics
}
