package internal_service

import (
	"context"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

type GetEndpointOption struct {
	InjectTag     bool
	InjectRetry   bool
	InjectCaching bool
}

func NewGetEndpointOption() *GetEndpointOption {
	return &GetEndpointOption{}
}

func NewDefaultGetEndpointOption() *GetEndpointOption {
	return &GetEndpointOption{
		InjectTag:     true,
		InjectRetry:   true,
		InjectCaching: true,
	}
}

type EndpointService interface {
	Get(ctx context.Context, auth types.SimplePrinciple,
		endpointId uint64, endpointProviderModelId *uint64, opts *GetEndpointOption) (*internal_gorm.Endpoint, error)

	GetAll(ctx context.Context,
		auth types.SimplePrinciple,
		criterias []*endpoint_grpc_api.Criteria, paginate *endpoint_grpc_api.Paginate) (int64, []*internal_gorm.Endpoint, error)

	GetAllEndpointProviderModel(ctx context.Context,
		auth types.SimplePrinciple,
		endpointId uint64, criterias []*endpoint_grpc_api.Criteria, paginate *endpoint_grpc_api.Paginate) (int64, []*internal_gorm.EndpointProviderModel, error)

	UpdateEndpointVersion(ctx context.Context,
		auth types.SimplePrinciple,
		endpointId, endpointProviderModelId uint64,
	) (*internal_gorm.Endpoint, error)

	CreateEndpoint(ctx context.Context,
		auth types.SimplePrinciple,
		name string,
		description *string,
		visibility *string,
		source *string,
		sourceIdentifier *uint64) (*internal_gorm.Endpoint, error)

	CreateEndpointProviderModel(
		ctx context.Context,
		auth types.SimplePrinciple,
		endpointId uint64,
		description string,
		providerId uint64,
		providerName string,
		promptRequest string,
		options []*endpoint_grpc_api.Metadata,
	) (*internal_gorm.EndpointProviderModel, error)

	AttachProviderModelToEndpoint(ctx context.Context,
		auth types.SimplePrinciple,
		endpointProviderModelId, endpointId uint64) (*internal_gorm.Endpoint, error)
	/*
		In order to configure retry
	*/
	ConfigureEndpointCaching(ctx context.Context,
		auth types.SimplePrinciple,
		endpointId uint64,
		caching internal_gorm.Cache,
		expiryInterval uint64,
		matchThreshold float32,
	) (*internal_gorm.EndpointCaching, error)

	/**/

	ConfigureEndpointRetry(ctx context.Context,
		auth types.SimplePrinciple,
		endpointId uint64,
		retry internal_gorm.Retry,
		maxAttempts uint64,
		delaySeconds uint64,
		exponentialBackoff bool,
		retryables []string,
	) (*internal_gorm.EndpointRetry, error)

	//
	CreateOrUpdateEndpointTag(ctx context.Context,
		auth types.SimplePrinciple,
		endpointId uint64,
		tags []string,
	) (*internal_gorm.EndpointTag, error)

	UpdateEndpointDetail(ctx context.Context,
		auth types.SimplePrinciple,
		endpointId uint64, name string, description *string) (*internal_gorm.Endpoint, error)
}
