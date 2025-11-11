package internal_log_service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	internal_service "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm/clause"
)

type endpointLogService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func NewEndpointLogService(logger commons.Logger, postgres connectors.PostgresConnector) internal_service.EndpointLogService {
	return &endpointLogService{
		logger:   logger,
		postgres: postgres,
	}
}

func (els *endpointLogService) CreateEndpointLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	source utils.RapidaSource,
	endpointId, endpointProviderModelId uint64,
	logId uint64,
	arguments, metadata, options map[string]interface{},
) (*internal_gorm.EndpointLog, error) {
	db := els.postgres.DB(ctx)
	endpointLog := &internal_gorm.EndpointLog{
		Source: source.Get(),
		Audited: gorm_models.Audited{
			Id: logId,
		},
		EndpointId:              endpointId,
		EndpointProviderModelId: endpointProviderModelId,
		Organizational: gorm_models.Organizational{
			ProjectId:      *auth.GetCurrentProjectId(),
			OrganizationId: *auth.GetCurrentOrganizationId(),
		},
		Status: type_enums.RECORD_IN_PROGRESS,
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"updated_date"}),
	}).Create(&endpointLog)
	if tx.Error != nil {
		return nil, tx.Error
	}

	utils.Go(ctx, func() {
		els.ApplyArgument(ctx, auth, logId, arguments)
	})
	utils.Go(ctx, func() {
		els.ApplyOption(ctx, auth, logId, options)

	})
	utils.Go(ctx, func() {
		els.ApplyMetadata(ctx, auth, logId, metadata)

	})
	return endpointLog, nil
}

func (els *endpointLogService) UpdateEndpointLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	logId uint64,
	metrics []*endpoint_grpc_api.Metric,
	timeTaken uint64,
) (*internal_gorm.EndpointLog, error) {
	db := els.postgres.DB(ctx)
	endpointLog := &internal_gorm.EndpointLog{
		Audited: gorm_models.Audited{
			Id: logId,
		},
		Organizational: gorm_models.Organizational{
			ProjectId:      *auth.GetCurrentProjectId(),
			OrganizationId: *auth.GetCurrentOrganizationId(),
		},
		Status:    type_enums.RECORD_COMPLETE,
		TimeTaken: timeTaken,
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"time_taken",
			"status",
			"updated_date"}),
	}).Create(&endpointLog)
	if tx.Error != nil {
		return nil, tx.Error
	}

	utils.Go(ctx, func() {
		els.ApplyMetrics(ctx, auth, logId, types.ToMetrics(metrics))
	})

	return endpointLog, nil
}

func (els *endpointLogService) ApplyMetadata(
	ctx context.Context,
	auth types.SimplePrinciple,
	logId uint64,
	metadata map[string]interface{},
) ([]*internal_gorm.EndpointLogMetadata, error) {
	start := time.Now()
	if len(metadata) == 0 {
		els.logger.Warnf("error while updating metadata, empty set of argument found")
		return nil, nil
	}
	db := els.postgres.DB(ctx)
	_metadatas := make([]*internal_gorm.EndpointLogMetadata, 0)
	//
	for k, mt := range metadata {
		_meta := &internal_gorm.EndpointLogMetadata{
			EndpointLogId: logId,
			Metadata: gorm_models.Metadata{
				Key: k,
			},
		}
		_meta.SetValue(mt)
		if auth.GetUserId() != nil {
			_meta.UpdatedBy = *auth.GetUserId()
			_meta.CreatedBy = *auth.GetUserId()
		}
		_metadatas = append(_metadatas, _meta)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "endpoint_log_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by", "updated_date"}),
	}).Create(&_metadatas)
	if tx.Error != nil {
		els.logger.Benchmark("els.ApplyMetadata", time.Since(start))
		els.logger.Errorf("error while ApplyMetadata %v", tx.Error)
		return nil, tx.Error
	}
	els.logger.Benchmark("els.ApplyMetadata", time.Since(start))
	return _metadatas, nil
}

func (els *endpointLogService) ApplyOption(ctx context.Context,
	auth types.SimplePrinciple,
	logId uint64,
	opts map[string]interface{}) ([]*internal_gorm.EndpointLogOption, error) {
	start := time.Now()
	if len(opts) == 0 {
		els.logger.Warnf("error while updating options, empty set of options found")
		return nil, nil
	}

	db := els.postgres.DB(ctx)
	options := make([]*internal_gorm.EndpointLogOption, 0)

	for k, o := range opts {
		option := &internal_gorm.EndpointLogOption{
			EndpointLogId: logId,
			Metadata: gorm_models.Metadata{
				Key: k,
			},
		}
		option.SetValue(o)
		if auth.GetUserId() != nil {
			option.CreatedBy = *auth.GetUserId()
			option.UpdatedBy = *auth.GetUserId()
		}
		options = append(options, option)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "endpoint_log_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by", "updated_date"}),
	}).Create(&options)
	if tx.Error != nil {
		els.logger.Benchmark("els.ApplyOption", time.Since(start))
		els.logger.Errorf("error while updating conversation options %v", tx.Error)
		return nil, tx.Error
	}
	els.logger.Benchmark("els.ApplyOption", time.Since(start))
	return options, nil

}

func (els *endpointLogService) ApplyArgument(ctx context.Context,
	auth types.SimplePrinciple,
	logId uint64,
	arguments map[string]interface{},
) ([]*internal_gorm.EndpointLogArgument, error) {
	start := time.Now()
	//
	if len(arguments) == 0 {
		els.logger.Warnf("error while updating arguments, empty set of argument found")
		return nil, nil
	}

	db := els.postgres.DB(ctx)
	_arguments := make([]*internal_gorm.EndpointLogArgument, 0)

	for k, arg := range arguments {
		ag := &internal_gorm.EndpointLogArgument{
			EndpointLogId: logId,
			Argument: gorm_models.Argument{
				Name: k,
			},
		}
		ag.SetValue(arg)
		if auth.GetUserId() != nil {
			ag.UpdatedBy = *auth.GetUserId()
		}
		_arguments = append(_arguments, ag)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "endpoint_log_id"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by", "updated_date"}),
	}).Create(&_arguments)
	if tx.Error != nil {
		els.logger.Benchmark("els.ApplyArgument", time.Since(start))
		els.logger.Errorf("error while updating conversation argument %v", tx.Error)
		return nil, tx.Error
	}
	els.logger.Benchmark("els.ApplyArgument", time.Since(start))
	return _arguments, nil
}

/**
* NOTE
* Feedback about the conversation
* Once the conversation is over the user will be prompted about conversation quality and xyz defined by the client
* client push the feedback as string and it will be stored as metrics later there might be different kind of feedback client can ask
**/
func (els *endpointLogService) ApplyMetrics(
	ctx context.Context,
	auth types.SimplePrinciple,
	logId uint64,
	metrics []*types.Metric,
) ([]*internal_gorm.EndpointLogMetric, error) {
	start := time.Now()
	db := els.postgres.DB(ctx)
	mtrs := make([]*internal_gorm.EndpointLogMetric, 0)
	for _, mtr := range metrics {
		_mtr := &internal_gorm.EndpointLogMetric{
			Metric: gorm_models.Metric{
				Name:        mtr.GetName(),
				Value:       mtr.GetValue(),
				Description: mtr.GetDescription(),
			},
			EndpointLogId: logId,
		}

		if auth.GetUserId() != nil {
			_mtr.UpdatedBy = *auth.GetUserId()
			_mtr.CreatedBy = *auth.GetUserId()
		}
		mtrs = append(mtrs, _mtr)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "endpoint_log_id"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value", "description",
			"updated_by", "updated_date"}),
	}).Create(&mtrs)
	if tx.Error != nil {
		els.logger.Benchmark("els.ApplyMetrics", time.Since(start))
		els.logger.Errorf("error while updating conversation %v", tx.Error)
		return nil, tx.Error
	}
	els.logger.Benchmark("els.ApplyMetrics", time.Since(start))
	return mtrs, nil
}

func (els *endpointLogService) GetAllEndpointLog(ctx context.Context,
	auth types.SimplePrinciple,
	endpointId uint64,
	criterias []*endpoint_grpc_api.Criteria, paginate *endpoint_grpc_api.Paginate) (int64, []*internal_gorm.EndpointLog, error) {
	start := time.Now()
	db := els.postgres.DB(ctx)
	var (
		endpointLogs []*internal_gorm.EndpointLog
		cnt          int64
	)
	qry := db.Model(internal_gorm.EndpointLog{})
	qry.
		Preload("Arguments").
		Preload("Metadata").
		Preload("Options").
		Preload("Metrics").
		Where("organization_id = ? AND project_id = ? AND endpoint_id = ?", *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId(), endpointId)
	for _, ct := range criterias {
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
		}).Find(&endpointLogs)

	if tx.Error != nil {
		els.logger.Errorf("not able to find any Webhooks %v", tx.Error)
		return cnt, nil, tx.Error
	}
	els.logger.Benchmark("EndpointLogService.GetAllLog", time.Since(start))
	return cnt, endpointLogs, nil
}
func (els *endpointLogService) GetEndpointLog(ctx context.Context, auth types.SimplePrinciple, logId, endpointId uint64) (*internal_gorm.EndpointLog, error) {
	start := time.Now()
	db := els.postgres.DB(ctx)
	var wkg *internal_gorm.EndpointLog
	tx := db.Where("id = ? AND organization_id = ? AND project_id = ? AND endpoint_id = ?", logId, *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId(), endpointId).
		First(&wkg)
	if tx.Error != nil {
		els.logger.Benchmark("EndpointLogService.GetLog", time.Since(start))
		els.logger.Errorf("not able to find any webhook %v", tx.Error)
		return nil, tx.Error
	}
	els.logger.Benchmark("EndpointLogService.GetLog", time.Since(start))
	return wkg, nil
}

func (els *endpointLogService) GetAggregatedEndpointAnalytics(ctx context.Context, auth types.SimplePrinciple, endpointId uint64) *endpoint_grpc_api.AggregatedEndpointAnalytics {
	criterias := []*endpoint_grpc_api.Criteria{{
		Key:   "created_date",
		Logic: ">=",
		Value: time.Now().AddDate(0, 0, -7).Format("2006-01-02 15:04:05"),
	}}

	count, logs, err := els.GetAllEndpointLog(
		ctx,
		auth,
		endpointId,
		criterias,
		&endpoint_grpc_api.Paginate{
			Page:     0,
			PageSize: 100,
		},
	)
	if err != nil {
		return &endpoint_grpc_api.AggregatedEndpointAnalytics{
			Count:        uint64(count),
			LastActivity: nil,
		}
	}

	var totalInputCost, totalOutputCost float32
	var totalToken, successCount, errorCount uint64
	var latencies []float32
	var lastActivity time.Time

	for _, log := range logs {
		for _, metric := range log.Metrics {
			value, err := strconv.ParseFloat(metric.Value, 64)
			if err != nil {
				continue
			}

			switch type_enums.MetricName(metric.Metric.Name) {
			case type_enums.INPUT_COST:
				totalInputCost += float32(value)
			case type_enums.OUTPUT_COST:
				totalOutputCost += float32(value)
			case type_enums.TOTAL_TOKEN:
				totalToken += uint64(value)
			case type_enums.TIME_TAKEN:
				latencies = append(latencies, float32(value))
			}
		}

		// Count successes and errors
		if log.Status == type_enums.RECORD_COMPLETE {
			successCount++
		} else {
			errorCount++
		}

		// Update last activity
		if time.Time(log.CreatedDate).After(lastActivity) {
			lastActivity = time.Time(log.CreatedDate)
		}
	}

	// Calculate percentiles
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	p50Index := int(float64(len(latencies)) * 0.5)
	p99Index := int(float64(len(latencies)) * 0.99)

	var p50Latency, p99Latency float32
	if len(latencies) > 0 {
		p50Latency = latencies[p50Index]
		p99Latency = latencies[p99Index]
	}

	return &endpoint_grpc_api.AggregatedEndpointAnalytics{
		Count:           uint64(count),
		TotalInputCost:  totalInputCost,
		TotalOutputCost: totalOutputCost,
		TotalToken:      totalToken,
		SuccessCount:    successCount,
		ErrorCount:      errorCount,
		P50Latency:      p50Latency,
		P99Latency:      p99Latency,
		LastActivity:    timestamppb.New(lastActivity),
	}

}
