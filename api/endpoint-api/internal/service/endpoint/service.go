package internal_endpoint_service

import (
	"context"
	"fmt"
	"time"

	"github.com/rapidaai/api/endpoint-api/config"
	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	internal_service "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

type endpointService struct {
	cfg      *config.EndpointConfig
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func NewEndpointService(cfg *config.EndpointConfig, logger commons.Logger, postgres connectors.PostgresConnector) internal_service.EndpointService {
	return &endpointService{
		cfg:      cfg,
		logger:   logger,
		postgres: postgres,
	}
}

func (eService *endpointService) Get(ctx context.Context,
	auth types.SimplePrinciple,
	endpointId uint64,
	endpointProviderModelId *uint64,
	opts *internal_service.GetEndpointOption) (*internal_gorm.Endpoint, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var endpoint *internal_gorm.Endpoint

	tx := db
	if opts.InjectCaching {
		tx.Preload("EndpointCaching")
	}
	if opts.InjectRetry {
		tx.Preload("EndpointRetry")
	}
	if opts.InjectTag {
		tx.Preload("EndpointTag")
	}

	if endpointProviderModelId != nil {
		tx = tx.
			Joins("inner join endpoint_provider_models EndpointProviderModel on EndpointProviderModel.endpoint_id = endpoints.id AND EndpointProviderModel.id = ?", endpointProviderModelId).
			Preload("EndpointProviderModel").
			Preload("EndpointProviderModel.EndpointProviderModelOptions")

	} else {
		tx = tx.
			Joins("inner join endpoint_provider_models EndpointProviderModel on EndpointProviderModel.endpoint_id = endpoints.id AND EndpointProviderModel.id = endpoints.endpoint_provider_model_id").
			Preload("EndpointProviderModel").
			Preload("EndpointProviderModel.EndpointProviderModelOptions")
	}
	tx = tx.
		Where("endpoints.id = ?", endpointId).
		First(&endpoint)
	if endpoint.Visibility != nil && *endpoint.Visibility != "public" {
		if *auth.GetCurrentOrganizationId() != endpoint.OrganizationId || *auth.GetCurrentProjectId() != endpoint.ProjectId {
			return nil, fmt.Errorf("you don't have access to the endpoint")
		}
	}

	if tx.Error != nil {
		eService.logger.Benchmark("endpointService.Get", time.Since(start))
		eService.logger.Errorf("not able to find any endpoint %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("endpointService.Get", time.Since(start))
	return endpoint, nil
}

// update endpoint version
func (eService *endpointService) UpdateEndpointVersion(ctx context.Context,
	auth types.SimplePrinciple,
	endpointId, endpointProviderModelId uint64) (*internal_gorm.Endpoint, error) {
	db := eService.postgres.DB(ctx)
	ed := &internal_gorm.Endpoint{
		EndpointProviderModelId: endpointProviderModelId,
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("id = ? AND project_id = ? AND organization_id = ?", endpointId,
		*auth.GetCurrentProjectId(),
		*auth.GetCurrentOrganizationId(),
	).Clauses(clause.Returning{}).Updates(ed)
	if tx.Error != nil {
		eService.logger.Errorf("error while updating endpoint %v", tx.Error)
		return nil, tx.Error
	}
	return ed, nil
}

func (eService *endpointService) GetAll(ctx context.Context, auth types.SimplePrinciple, criterias []*endpoint_grpc_api.Criteria, paginate *endpoint_grpc_api.Paginate) (int64, []*internal_gorm.Endpoint, error) {
	db := eService.postgres.DB(ctx)
	var (
		endpoints []*internal_gorm.Endpoint
		cnt       int64
	)
	qry := db.Debug().Model(internal_gorm.Endpoint{})
	qry.
		Preload("EndpointTag").
		Preload("EndpointRetry").
		Preload("EndpointCaching").
		Preload("EndpointProviderModel").
		Where("organization_id = ? AND project_id = ? AND status = ?", *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId(), type_enums.RECORD_ACTIVE.String())
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
		}).Find(&endpoints)

	if tx.Error != nil {
		eService.logger.Errorf("not able to find any endpoint %v", tx.Error)
		return cnt, nil, tx.Error
	}

	return cnt, endpoints, nil
}
func (eService *endpointService) GetAllEndpointProviderModel(ctx context.Context, auth types.SimplePrinciple, endpointId uint64, criterias []*endpoint_grpc_api.Criteria, paginate *endpoint_grpc_api.Paginate) (int64, []*internal_gorm.EndpointProviderModel, error) {
	db := eService.postgres.DB(ctx)
	var (
		epms []*internal_gorm.EndpointProviderModel
		cnt  int64
	)
	// use projectId and orgId to validate that he has access to the endpoint
	qry := db.Model(internal_gorm.EndpointProviderModel{})
	qry.
		Preload("EndpointProviderModelOptions").
		Where("endpoint_id = ? ", endpointId)
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
		}).Find(&epms)

	if tx.Error != nil {
		eService.logger.Errorf("not able to find any endpoint %v", tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, epms, nil
}
func (eService *endpointService) CreateEndpoint(ctx context.Context,
	auth types.SimplePrinciple,
	name string,
	description *string,
	visibility *string,
	source *string,
	sourceIdentifier *uint64,
) (*internal_gorm.Endpoint, error) {
	db := eService.postgres.DB(ctx)
	ep := &internal_gorm.Endpoint{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: *auth.GetUserId(),
		},
		Organizational: gorm_models.Organizational{
			ProjectId:      *auth.GetCurrentProjectId(),
			OrganizationId: *auth.GetCurrentOrganizationId(),
		},
		Name:       name,
		Visibility: utils.Ptr("private"),
	}

	if description != nil {
		ep.Description = description
	}

	if visibility != nil {
		ep.Visibility = visibility
	}

	if source != nil {
		ep.Source = source
	}

	if sourceIdentifier != nil {
		ep.SourceIdentifier = sourceIdentifier
	}

	if err := db.Save(ep).Error; err != nil {
		eService.logger.Errorf("unable to create endpoint with error %+v", err)
		return nil, err
	}
	return ep, nil

}

func (eService *endpointService) CreateEndpointProviderModel(
	ctx context.Context,
	auth types.SimplePrinciple,
	endpointId uint64,
	description string,
	providerId uint64,
	providerName string,
	promptRequest string,
	options []*endpoint_grpc_api.Metadata,
) (*internal_gorm.EndpointProviderModel, error) {

	db := eService.postgres.DB(ctx)
	epm := &internal_gorm.EndpointProviderModel{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: *auth.GetUserId(),
		},
		ModelProviderId:   providerId,
		Description:       description,
		ModelProviderName: providerName,
		EndpointId:        endpointId,
	}
	epm.SetPrompt(promptRequest)
	tx := db.Save(epm)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create endpoint.")
		return nil, tx.Error
	}

	if len(options) == 0 {
		return epm, nil
	}
	modelOptions := make([]*internal_gorm.EndpointProviderModelOption, 0)
	for _, v := range options {
		modelOptions = append(modelOptions, &internal_gorm.EndpointProviderModelOption{
			EndpointProviderModelId: epm.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RECORD_ACTIVE,
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}
	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "endpoint_provider_model_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(modelOptions)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create deployment audio config metadata for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	return epm, nil
}

func (eService *endpointService) AttachProviderModelToEndpoint(ctx context.Context,
	auth types.SimplePrinciple,
	endpointProviderModelId, endpointId uint64) (*internal_gorm.Endpoint, error) {
	db := eService.postgres.DB(ctx)
	ed := &internal_gorm.Endpoint{
		EndpointProviderModelId: endpointProviderModelId,
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("id = ? AND project_id = ? AND organization_id = ?", endpointId,
		*auth.GetCurrentProjectId(),
		*auth.GetCurrentOrganizationId(),
	).Clauses(clause.Returning{}).Updates(ed)
	if tx.Error != nil {
		eService.logger.Errorf("error while updating for endpoint provider model %v", tx.Error)
		return nil, tx.Error
	}
	return ed, nil
}

/*
Configuring endpoint retry
*/
func (eService *endpointService) ConfigureEndpointRetry(ctx context.Context,
	auth types.SimplePrinciple,
	endpointId uint64,
	retry internal_gorm.Retry,
	maxAttempts uint64,
	delaySeconds uint64,
	exponentialBackoff bool,
	retryables []string,
) (*internal_gorm.EndpointRetry, error) {
	db := eService.postgres.DB(ctx)

	retryEnable := retry != internal_gorm.NEVER_RETRY
	endpoint := &internal_gorm.Endpoint{
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
		RetryEnable: retryEnable,
	}
	tx := db.Where("id = ?", endpointId).
		Clauses(clause.Returning{}).
		Updates(endpoint)
	if tx.Error != nil {
		eService.logger.Errorf("error while updating for endpoint configuration %v", tx.Error)
		return nil, tx.Error
	}

	retryEndpoint := &internal_gorm.EndpointRetry{
		EndpointId:         endpointId,
		RetryType:          retry,
		MaxAttempts:        maxAttempts,
		DelaySeconds:       delaySeconds,
		ExponentialBackoff: exponentialBackoff,
		Retryables:         retryables,
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "endpoint_id"}, {Name: "retry_type"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"max_attempts",
			"delay_seconds", "exponential_backoff", "retryables"}),
	}).Create(&retryEndpoint)

	if tx.Error != nil {
		eService.logger.Errorf("error while updating retry configuration %v", tx.Error)
		return nil, tx.Error
	}
	return retryEndpoint, nil
}

/*
Configuring endpoint retry
*/
func (eService *endpointService) ConfigureEndpointCaching(ctx context.Context,
	auth types.SimplePrinciple,
	endpointId uint64,
	caching internal_gorm.Cache,
	expiryInterval uint64,
	matchThreshold float32,
) (*internal_gorm.EndpointCaching, error) {
	db := eService.postgres.DB(ctx)
	cacheEnable := caching != internal_gorm.NEVER_CACHE
	endpoint := &internal_gorm.Endpoint{
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
		CacheEnable: cacheEnable,
	}
	tx := db.Where("id = ?", endpointId).
		Clauses(clause.Returning{}).
		Updates(endpoint)
	if tx.Error != nil {
		eService.logger.Errorf("error while updating for endpoint configuration %v", tx.Error)
		return nil, tx.Error
	}

	cachingEndpoint := &internal_gorm.EndpointCaching{
		EndpointId:     endpointId,
		CacheType:      caching,
		ExpiryInterval: expiryInterval,
		MatchThreshold: matchThreshold,
		CreatedBy:      *auth.GetUserId(),
		UpdatedBy:      *auth.GetUserId(),
	}
	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "endpoint_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"cache_type",
			"expiry_interval", "match_threshold"}),
	}).Create(&cachingEndpoint)

	if tx.Error != nil {
		eService.logger.Errorf("error while updating retry configuration %v", tx.Error)
		return nil, tx.Error
	}
	return cachingEndpoint, nil
}

func (eService *endpointService) CreateOrUpdateEndpointTag(ctx context.Context,
	auth types.SimplePrinciple,
	endpointId uint64,
	tags []string,
) (*internal_gorm.EndpointTag, error) {

	db := eService.postgres.DB(ctx)
	endpointTag := &internal_gorm.EndpointTag{
		EndpointId: endpointId,
		Tag:        tags,
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "endpoint_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"tag",
			"updated_by"}),
	}).Create(&endpointTag)

	if tx.Error != nil {
		eService.logger.Errorf("error while updating tags %v", tx.Error)
		return nil, tx.Error
	}
	return endpointTag, nil
}

func (eService *endpointService) UpdateEndpointDetail(ctx context.Context,
	auth types.SimplePrinciple,
	endpointId uint64, name string, description *string) (*internal_gorm.Endpoint, error) {
	db := eService.postgres.DB(ctx)
	ed := &internal_gorm.Endpoint{
		Name:        name,
		Description: description,
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
	}

	tx := db.Where("id = ? AND project_id = ? AND organization_id = ?", endpointId,
		*auth.GetCurrentProjectId(),
		*auth.GetCurrentOrganizationId(),
	).Updates(ed)
	if tx.Error != nil {
		eService.logger.Errorf("error while updating for endpoint %v", tx.Error)
		return nil, tx.Error
	}
	return ed, nil

}
