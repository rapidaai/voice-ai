package internal_service

import (
	"context"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
	protos "github.com/rapidaai/protos"
)

type EndpointLogService interface {
	CreateEndpointLog(
		ctx context.Context,
		auth *types.Authentication,
		source utils.RapidaSource,
		endpointId, endpointProviderModelId uint64,
		logId uint64,
		arguments, metadata, options map[string]interface{},
	) (*internal_gorm.EndpointLog, error)
	UpdateEndpointLog(
		ctx context.Context,
		auth *types.Authentication,
		logId uint64,
		metrics []*protos.Metric,
		timeTaken uint64,
	) (*internal_gorm.EndpointLog, error)

	GetAllEndpointLog(ctx context.Context,
		auth *types.Authentication,
		endpointId uint64,
		criteria []*endpoint_grpc_api.Criteria, paginate *endpoint_grpc_api.Paginate) (int64, []*internal_gorm.EndpointLog, error)
	GetEndpointLog(ctx context.Context, auth *types.Authentication, logId, endpointId uint64) (*internal_gorm.EndpointLog, error)
	GetAggregatedEndpointAnalytics(ctx context.Context, auth *types.Authentication, endpointId uint64) *protos.AggregatedEndpointAnalytics
}
